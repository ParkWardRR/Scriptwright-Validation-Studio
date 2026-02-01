package runner

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"philadelphia/internal/userscript"

	"github.com/playwright-community/playwright-go"
)

// Options configure a run.
type Options struct {
	TargetURL           string
	ScriptPath          string // optional if ScriptContent set
	ScriptContent       string
	ScriptURL           string // optional remote fetch
	ScriptGitRepo       string // optional git repo URL
	ScriptGitPath       string // path inside git repo
	ExtensionDir        string // optional: path to unpacked MV3 extension (e.g., Tampermonkey)
	Engine              string // display only for now
	Headless            bool
	ProfileDir          string // optional persistent profile location
	CaptureTrace        bool
	CaptureHAR          bool
	ReplayHAR           string   // optional path to HAR for replay
	BaselineDir         string   // for visual regression hashes
	VisualDiffThreshold float64  // per-channel threshold 0-255
	BlockedHosts        []string // basic network assertion
	Steps               []Step   // flow actions/assertions
	Workspace           string   // base path; defaults to cwd
}

// Step represents a simple flow action or assertion.
type Step struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
	Value  string `json:"value,omitempty"`
	Assert string `json:"assert,omitempty"` // e.g., "text-equals", "contains", "exists", "not-exists", "attr"
	Attr   string `json:"attr,omitempty"`   // used with assert attr
}

// Result contains artifact paths and manifest.
type Result struct {
	RunID     string
	RunDir    string
	Manifest  Manifest
	LogPath   string
	Artifacts struct {
		Screenshot string
		VideoWebM  string
		VideoWebP  string
		TraceZip   string
		HAR        string
	}
}

// Manifest is persisted to run.json.
type Manifest struct {
	RunID            string          `json:"run_id"`
	StartedAt        time.Time       `json:"started_at"`
	FinishedAt       time.Time       `json:"finished_at"`
	TargetURL        string          `json:"target_url"`
	Screenshot       string          `json:"screenshot"`
	VideoWebM        string          `json:"video_webm,omitempty"`
	VideoWebP        string          `json:"video_webp,omitempty"`
	TraceZip         string          `json:"trace_zip,omitempty"`
	HAR              string          `json:"har,omitempty"`
	ReplayHAR        string          `json:"replay_har,omitempty"`
	ScriptMeta       userscript.Meta `json:"script_meta"`
	ProfileFolder    string          `json:"profile_folder"`
	Engine           string          `json:"engine"`
	ExtensionDir     string          `json:"extension_dir,omitempty"`
	LogPath          string          `json:"log_path"`
	VisualHash       string          `json:"visual_hash,omitempty"`
	VisualDiff       bool            `json:"visual_diff,omitempty"`
	VisualDiffImg    string          `json:"visual_diff_img,omitempty"`
	VisualDiffPixels int             `json:"visual_diff_pixels,omitempty"`
	VisualDiffRatio  float64         `json:"visual_diff_ratio,omitempty"`
	NetworkIssues    []string        `json:"network_issues,omitempty"`
}

// Run executes a single userscript against a URL and produces artifacts.
func Run(opts Options) (Result, error) {
	if opts.TargetURL == "" {
		return Result{}, errors.New("TargetURL is required")
	}
	if opts.ScriptPath == "" {
		return Result{}, errors.New("ScriptPath is required")
	}
	if opts.Workspace == "" {
		cwd, _ := os.Getwd()
		opts.Workspace = cwd
	}
	if opts.ScriptPath == "" && opts.ScriptContent == "" && opts.ScriptURL == "" && opts.ScriptGitRepo == "" {
		return Result{}, errors.New("provide ScriptPath, ScriptContent, ScriptURL, or ScriptGitRepo")
	}
	if opts.ScriptPath == "" && opts.ScriptContent != "" {
		tmp, err := os.CreateTemp("", "userscript-*.user.js")
		if err != nil {
			return Result{}, err
		}
		if _, err := tmp.WriteString(opts.ScriptContent); err != nil {
			return Result{}, err
		}
		tmp.Close()
		opts.ScriptPath = tmp.Name()
	}
	if opts.ScriptPath == "" && opts.ScriptURL != "" {
		tmp, err := fetchScript(opts.ScriptURL)
		if err != nil {
			return Result{}, fmt.Errorf("fetch script: %w", err)
		}
		opts.ScriptPath = tmp
	}
	if opts.ScriptPath == "" && opts.ScriptGitRepo != "" {
		tmp, err := fetchScriptFromGit(opts.ScriptGitRepo, opts.ScriptGitPath)
		if err != nil {
			return Result{}, fmt.Errorf("git fetch: %w", err)
		}
		opts.ScriptPath = tmp
	}

	runID := fmt.Sprintf("%x", time.Now().UnixNano())
	runDir := filepath.Join(opts.Workspace, "runs", runID)
	artifactsDir := filepath.Join(runDir, "artifacts")
	logsDir := filepath.Join(runDir, "logs")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return Result{}, err
	}

	logPath := filepath.Join(logsDir, "runner.ndjson")
	logFile, err := os.Create(logPath)
	if err != nil {
		return Result{}, err
	}
	defer logFile.Close()
	logger := newNDJSONLogger(logFile)

	scriptMeta, err := userscript.Parse(opts.ScriptPath)
	if err != nil {
		return Result{}, fmt.Errorf("parse userscript: %w", err)
	}
	scriptContent, err := os.ReadFile(opts.ScriptPath)
	if err != nil {
		return Result{}, err
	}

	logger.info("runner", "installing playwright browsers", nil)
	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}}); err != nil {
		return Result{}, err
	}

	pw, err := playwright.Run()
	if err != nil {
		return Result{}, err
	}
	defer pw.Stop()

	profileDir := opts.ProfileDir
	if profileDir == "" {
		profileDir = filepath.Join(os.TempDir(), "philadelphia-profile-"+runID)
	}
	_ = os.RemoveAll(profileDir)

	ctxOpts := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(opts.Headless),
		Args:     []string{"--disable-dev-shm-usage"},
		RecordVideo: &playwright.RecordVideo{
			Dir:  filepath.Join(artifactsDir, "video"),
			Size: &playwright.Size{Width: 1280, Height: 720},
		},
	}
	if opts.ExtensionDir != "" {
		ctxOpts.Args = append(ctxOpts.Args,
			"--disable-extensions-except="+opts.ExtensionDir,
			"--load-extension="+opts.ExtensionDir,
		)
		logger.info("runner", "attempting MV3 extension load", map[string]any{"extension_dir": opts.ExtensionDir})
	}
	if opts.CaptureHAR {
		harPath := filepath.Join(artifactsDir, "network.har")
		ctxOpts.RecordHarPath = playwright.String(harPath)
		ctxOpts.RecordHarURLFilter = "*"
	}

	ctx, err := pw.Chromium.LaunchPersistentContext(profileDir, ctxOpts)
	if err != nil {
		return Result{}, fmt.Errorf("launch context: %w", err)
	}
	defer ctx.Close()

	if opts.ExtensionDir != "" {
		if extID := detectExtensionID(ctx); extID != "" {
			logger.info("extension", "detected extension id", map[string]any{"id": extID})
		} else {
			logger.warn("extension", "could not detect extension id from service workers", nil)
		}
	}

	if opts.ReplayHAR != "" {
		if err := ctx.RouteFromHAR(opts.ReplayHAR); err != nil {
			logger.warn("har", "route from HAR failed", map[string]any{"error": err.Error()})
		} else {
			logger.info("har", "replaying from HAR", map[string]any{"path": opts.ReplayHAR})
		}
	}

	var responses []playwright.Response
	ctx.OnResponse(func(resp playwright.Response) {
		responses = append(responses, resp)
	})

	page, err := ctx.NewPage()
	if err != nil {
		return Result{}, err
	}

	// Inject script pre-navigation to approximate engine execution.
	engineLower := strings.ToLower(opts.Engine)
	installed := false
	if strings.Contains(engineLower, "tampermonkey") {
		installed = installTampermonkey(ctx, opts.ScriptPath, logger)
	}
	if !installed {
		if err := page.AddInitScript(playwright.Script{Content: playwright.String(string(scriptContent))}); err != nil {
			logger.warn("runner", "init script injection failed; continuing", map[string]any{"error": err.Error()})
		}
	}

	start := time.Now()
	logger.info("browser", "navigating", map[string]any{"url": opts.TargetURL})
	if _, err := page.Goto(opts.TargetURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(40_000),
	}); err != nil {
		return Result{}, fmt.Errorf("navigate: %w", err)
	}

	// Execute flow steps or default toggle.
	if len(opts.Steps) > 0 {
		executeSteps(page, opts.Steps, logger)
	} else {
		if _, err := page.WaitForSelector("text=Toggle Dark Mode", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(8_000),
		}); err != nil {
			logger.warn("assert", "toggle button not found", map[string]any{"error": err.Error()})
		} else {
			logger.info("assert", "toggle button present", nil)
		}
		if err := page.Click("text=Toggle Dark Mode"); err != nil {
			logger.warn("action", "click toggle failed", map[string]any{"error": err.Error()})
		} else {
			logger.info("action", "toggled dark mode", nil)
		}
	}

	page.WaitForTimeout(1200)

	screenshotPath := filepath.Join(artifactsDir, "screenshot.png")
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	}); err != nil {
		logger.warn("artifact", "screenshot failed", map[string]any{"error": err.Error()})
	}

	visualHash, visualDiff, visualDiffImg, visualDiffPixels, visualDiffRatio := computeVisualHash(screenshotPath, opts, artifactsDir, logger)

	video := page.Video()
	if err := page.Close(); err != nil {
		logger.warn("runner", "close page", map[string]any{"error": err.Error()})
	}

	if opts.CaptureTrace {
		tracePath := filepath.Join(artifactsDir, "trace.zip")
		if err := ctx.Tracing().Start(playwright.TracingStartOptions{Screenshots: playwright.Bool(true), Snapshots: playwright.Bool(true), Sources: playwright.Bool(true)}); err != nil {
			logger.warn("trace", "start failed", map[string]any{"error": err.Error()})
		}
		defer func() {
			if err := ctx.Tracing().StopChunk(tracePath); err != nil {
				logger.warn("trace", "stop failed", map[string]any{"error": err.Error()})
			} else {
				logger.info("trace", "trace captured", map[string]any{"path": tracePath})
			}
		}()
	}

	if opts.CaptureHAR {
		// TODO: integrate HAR recording
	}

	var videoPath, webpPath string
	if video != nil {
		videoPath, err = video.Path()
		if err != nil {
			logger.warn("artifact", "video path error", map[string]any{"error": err.Error()})
		}
	}
	if videoPath != "" {
		webpPath = filepath.Join(artifactsDir, "run.webp")
		if err := convertToWebP(videoPath, webpPath); err != nil {
			logger.warn("artifact", "webp conversion failed", map[string]any{"error": err.Error()})
			webpPath = ""
		} else {
			logger.info("artifact", "webp created", map[string]any{"path": webpPath})
		}
	}

	if err := ctx.Close(); err != nil {
		logger.warn("runner", "close context", map[string]any{"error": err.Error()})
	}

	manifest := Manifest{
		RunID:            runID,
		StartedAt:        start,
		FinishedAt:       time.Now(),
		TargetURL:        opts.TargetURL,
		Screenshot:       filepath.Base(screenshotPath),
		VisualHash:       visualHash,
		VisualDiff:       visualDiff,
		VisualDiffImg:    visualDiffImg,
		VisualDiffPixels: visualDiffPixels,
		VisualDiffRatio:  visualDiffRatio,
		VideoWebM:        filepath.Base(videoPath),
		VideoWebP:        filepath.Base(webpPath),
		TraceZip:         traceNameIfExists(artifactsDir),
		HAR:              harNameIfExists(artifactsDir),
		ReplayHAR:        opts.ReplayHAR,
		ScriptMeta:       scriptMeta,
		ProfileFolder:    profileDir,
		Engine:           opts.Engine,
		ExtensionDir:     opts.ExtensionDir,
		LogPath:          logPath,
		NetworkIssues:    summarizeNetwork(responses, opts.BlockedHosts, logger),
	}

	manifestPath := filepath.Join(runDir, "run.json")
	if err := writeManifest(manifestPath, manifest); err != nil {
		logger.warn("runner", "write manifest failed", map[string]any{"error": err.Error()})
	}

	logger.info("runner", "run finished", map[string]any{"run_id": runID})

	res := Result{
		RunID:    runID,
		RunDir:   runDir,
		Manifest: manifest,
		LogPath:  logPath,
	}
	res.Artifacts.Screenshot = filepath.Join("runs", runID, "artifacts", filepath.Base(screenshotPath))
	if videoPath != "" {
		res.Artifacts.VideoWebM = filepath.Join("runs", runID, "artifacts", filepath.Base(videoPath))
	}
	if webpPath != "" {
		res.Artifacts.VideoWebP = filepath.Join("runs", runID, "artifacts", filepath.Base(webpPath))
	}
	return res, nil
}

// --- helpers ---

type ndjsonLogger struct {
	w *bufio.Writer
}

type logLine struct {
	TS    time.Time      `json:"ts"`
	Level string         `json:"level"`
	Scope string         `json:"scope"`
	Msg   string         `json:"msg"`
	Meta  map[string]any `json:"meta,omitempty"`
}

func newNDJSONLogger(file *os.File) *ndjsonLogger {
	return &ndjsonLogger{w: bufio.NewWriter(file)}
}

func (l *ndjsonLogger) write(level, scope, msg string, meta map[string]any) {
	line := logLine{TS: time.Now(), Level: level, Scope: scope, Msg: msg, Meta: meta}
	b, _ := json.Marshal(line)
	l.w.Write(b)
	l.w.WriteByte('\n')
	l.w.Flush()
}

func (l *ndjsonLogger) info(scope, msg string, meta map[string]any) {
	l.write("info", scope, msg, meta)
}
func (l *ndjsonLogger) warn(scope, msg string, meta map[string]any) {
	l.write("warn", scope, msg, meta)
}

func convertToWebP(input, output string) error {
	if input == "" {
		return errors.New("empty input video path")
	}
	if err := tryFfmpegWebp(input, output); err == nil {
		return nil
	}
	framesDir, err := os.MkdirTemp("", "webp-frames")
	if err != nil {
		return err
	}
	defer os.RemoveAll(framesDir)

	framePattern := filepath.Join(framesDir, "frame-%03d.png")
	cmd := exec.Command("ffmpeg", "-y", "-i", input, "-vf", "fps=15,scale=1280:-1:flags=lanczos", framePattern)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	frames, err := filepath.Glob(filepath.Join(framesDir, "frame-*.png"))
	if err != nil {
		return err
	}
	if len(frames) == 0 {
		return errors.New("no frames generated for webp conversion")
	}

	args := append([]string{"-loop", "0"}, frames...)
	args = append(args, "-o", output)
	webpCmd := exec.Command("img2webp", args...)
	webpCmd.Stdout = os.Stdout
	webpCmd.Stderr = os.Stderr
	return webpCmd.Run()
}

func tryFfmpegWebp(input, output string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", input, "-vcodec", "libwebp", "-filter:v", "fps=15,scale=1280:-1:flags=lanczos", "-loop", "0", "-an", "-fps_mode", "cfr", output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeManifest(path string, manifest Manifest) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(manifest)
}

// LoadManifest reads a manifest from disk.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// FindRuns returns run directories under workspace/runs.
func FindRuns(workspace string) ([]string, error) {
	runsDir := filepath.Join(workspace, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// GuessWorkspace tries to pick a reasonable workspace root.
func GuessWorkspace() string {
	cwd, _ := os.Getwd()
	return cwd
}

// DiscoverExtensionDir returns the path from environment if set.
func DiscoverExtensionDir() string {
	ext := os.Getenv("USERSCRIPT_ENGINE_EXT_DIR")
	return strings.TrimSpace(ext)
}

// computeVisualHash creates a SHA256 of the screenshot and compares to any baseline.
func computeVisualHash(screenshotPath string, opts Options, artifactsDir string, logger *ndjsonLogger) (hash string, diff bool, diffImg string, diffPixels int, diffRatio float64) {
	data, err := os.ReadFile(screenshotPath)
	if err != nil || len(data) == 0 {
		return "", false, "", 0, 0
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	hash = sum
	if opts.BaselineDir == "" {
		return hash, false, "", 0, 0
	}
	if err := os.MkdirAll(opts.BaselineDir, 0o755); err != nil {
		logger.warn("visual", "baseline dir create failed", map[string]any{"error": err.Error()})
		return hash, false, "", 0, 0
	}
	basePath := filepath.Join(opts.BaselineDir, "screenshot.png")
	if _, err := os.Stat(basePath); err != nil {
		_ = os.WriteFile(basePath, data, 0o644)
		logger.info("visual", "baseline created", map[string]any{"path": basePath})
		return hash, false, "", 0, 0
	}
	baseData, err := os.ReadFile(basePath)
	if err != nil {
		logger.warn("visual", "baseline read failed", map[string]any{"error": err.Error()})
		return hash, false, "", 0, 0
	}
	baseHash := fmt.Sprintf("%x", sha256.Sum256(baseData))
	if baseHash != hash {
		logger.warn("visual", "screenshot hash mismatch vs baseline", map[string]any{"baseline": baseHash, "current": hash})
		diffImg, diffPixels, diffRatio = computeDiffImage(baseData, data, artifactsDir, logger, opts.VisualDiffThreshold)
		return hash, true, diffImg, diffPixels, diffRatio
	}
	logger.info("visual", "screenshot matches baseline", nil)
	return hash, false, "", 0, 0
}

// computeDiffImage generates a simple diff heatmap (red overlay) when sizes match.
func computeDiffImage(basePNG, currentPNG []byte, artifactsDir string, logger *ndjsonLogger, threshold float64) (string, int, float64) {
	baseImg, err := png.Decode(bytes.NewReader(basePNG))
	if err != nil {
		logger.warn("visual", "decode baseline failed", map[string]any{"error": err.Error()})
		return "", 0, 0
	}
	currImg, err := png.Decode(bytes.NewReader(currentPNG))
	if err != nil {
		logger.warn("visual", "decode current failed", map[string]any{"error": err.Error()})
		return "", 0, 0
	}
	if baseImg.Bounds() != currImg.Bounds() {
		logger.warn("visual", "baseline/current size mismatch", nil)
		return "", 0, 0
	}
	bounds := baseImg.Bounds()
	diff := image.NewRGBA(bounds)
	var changed int
	total := (bounds.Max.X - bounds.Min.X) * (bounds.Max.Y - bounds.Min.Y)
	thr := threshold
	if thr < 0 {
		thr = 0
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			br, bg, bb, _ := baseImg.At(x, y).RGBA()
			cr, cg, cb, _ := currImg.At(x, y).RGBA()
			dr := absDiff(br, cr)
			dg := absDiff(bg, cg)
			db := absDiff(bb, cb)
			if float64(dr) > thr || float64(dg) > thr || float64(db) > thr {
				changed++
				diff.Set(x, y, color.RGBA{R: 255, A: 180})
			} else {
				diff.Set(x, y, color.RGBA{A: 0})
			}
		}
	}
	if changed == 0 {
		return "", 0, 0
	}
	out := filepath.Join(artifactsDir, "visual-diff.png")
	f, err := os.Create(out)
	if err != nil {
		logger.warn("visual", "write diff failed", map[string]any{"error": err.Error()})
		return "", changed, float64(changed) / float64(total)
	}
	defer f.Close()
	if err := png.Encode(f, diff); err != nil {
		logger.warn("visual", "encode diff failed", map[string]any{"error": err.Error()})
		return "", changed, float64(changed) / float64(total)
	}
	logger.warn("visual", "diff image generated", map[string]any{"path": out, "pixels_changed": changed})
	return filepath.Base(out), changed, float64(changed) / float64(total)
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
func harNameIfExists(artifactsDir string) string {
	har := filepath.Join(artifactsDir, "network.har")
	if _, err := os.Stat(har); err == nil {
		return filepath.Base(har)
	}
	return ""
}

func traceNameIfExists(artifactsDir string) string {
	trace := filepath.Join(artifactsDir, "trace.zip")
	if _, err := os.Stat(trace); err == nil {
		return filepath.Base(trace)
	}
	return ""
}

func summarizeNetwork(responses []playwright.Response, blocked []string, logger *ndjsonLogger) []string {
	var issues []string
	for _, r := range responses {
		status := r.Status()
		url := r.URL()
		if status >= 400 {
			msg := fmt.Sprintf("status %d for %s", status, url)
			issues = append(issues, msg)
		}
		for _, host := range blocked {
			if strings.Contains(url, host) {
				msg := fmt.Sprintf("blocked host seen: %s", url)
				issues = append(issues, msg)
			}
		}
	}
	if len(issues) > 0 {
		logger.warn("network", "issues detected", map[string]any{"count": len(issues)})
	}
	return issues
}

func detectExtensionID(ctx playwright.BrowserContext) string {
	for _, sw := range ctx.ServiceWorkers() {
		url := sw.URL()
		parts := strings.Split(url, "/")
		for i, p := range parts {
			if p == "chrome-extension:" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
		if strings.HasPrefix(url, "chrome-extension://") {
			tokens := strings.Split(strings.TrimPrefix(url, "chrome-extension://"), "/")
			if len(tokens) > 0 {
				return tokens[0]
			}
		}
	}
	return ""
}

// installTampermonkey attempts deterministic install of the provided userscript into TM MV3.
// For now it opens the internal userscript.html import page and drops the file via file chooser.
func installTampermonkey(ctx playwright.BrowserContext, scriptPath string, logger *ndjsonLogger) bool {
	page, err := ctx.NewPage()
	if err != nil {
		logger.warn("tm", "new page failed", map[string]any{"error": err.Error()})
		return false
	}
	localURL := "chrome-extension://ddddjjjklioejkhhafmeepjlcenalaol/userscript.html"
	if _, err := page.Goto(localURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle}); err != nil {
		logger.warn("tm", "open userscript.html failed", map[string]any{"error": err.Error()})
		return false
	}
	input, err := page.QuerySelector("input[type=file]")
	if err != nil || input == nil {
		logger.warn("tm", "file input not found", map[string]any{"error": err})
		return false
	}
	if err := input.SetInputFiles([]string{scriptPath}); err != nil {
		logger.warn("tm", "set file failed", map[string]any{"error": err.Error()})
		return false
	}
	page.WaitForTimeout(1200)
	logger.info("tm", "script dropped into TM import", nil)
	return true
}

// executeSteps runs a minimal action/assertion DSL against the page.
func executeSteps(page playwright.Page, steps []Step, logger *ndjsonLogger) {
	for i, step := range steps {
		scope := fmt.Sprintf("step-%d", i+1)
		switch strings.ToLower(step.Action) {
		case "click":
			if err := page.Click(step.Target); err != nil {
				logger.warn(scope, "click failed", map[string]any{"error": err.Error(), "target": step.Target})
			} else {
				logger.info(scope, "click ok", map[string]any{"target": step.Target})
			}
		case "fill":
			if err := page.Fill(step.Target, step.Value); err != nil {
				logger.warn(scope, "fill failed", map[string]any{"error": err.Error(), "target": step.Target})
			} else {
				logger.info(scope, "fill ok", map[string]any{"target": step.Target})
			}
		case "waitforselector":
			if _, err := page.WaitForSelector(step.Target, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(8000)}); err != nil {
				logger.warn(scope, "waitForSelector failed", map[string]any{"error": err.Error(), "target": step.Target})
			} else {
				logger.info(scope, "selector present", map[string]any{"target": step.Target})
			}
		case "wait":
			d := 500.0
			if v, err := strconv.ParseFloat(step.Value, 64); err == nil && v > 0 {
				d = v
			}
			page.WaitForTimeout(d)
			logger.info(scope, "waited", map[string]any{"ms": d})
		case "assert-text", "assert-equals":
			text, err := page.TextContent(step.Target)
			if err != nil {
				logger.warn(scope, "assert-text failed", map[string]any{"error": err.Error(), "target": step.Target})
				continue
			}
			got := strings.TrimSpace(text)
			if got != step.Value {
				logger.warn(scope, "assert-text mismatch", map[string]any{"target": step.Target, "expected": step.Value, "got": got})
			} else {
				logger.info(scope, "assert-text ok", map[string]any{"target": step.Target, "value": got})
			}
		case "assert-contains":
			text, err := page.TextContent(step.Target)
			if err != nil {
				logger.warn(scope, "assert-contains failed", map[string]any{"error": err.Error(), "target": step.Target})
				continue
			}
			got := strings.TrimSpace(text)
			if !strings.Contains(got, step.Value) {
				logger.warn(scope, "assert-contains mismatch", map[string]any{"target": step.Target, "expected_substring": step.Value, "got": got})
			} else {
				logger.info(scope, "assert-contains ok", map[string]any{"target": step.Target, "value": got})
			}
		case "assert-exists":
			if _, err := page.WaitForSelector(step.Target, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(5000)}); err != nil {
				logger.warn(scope, "assert-exists failed", map[string]any{"error": err.Error(), "target": step.Target})
			} else {
				logger.info(scope, "assert-exists ok", map[string]any{"target": step.Target})
			}
		case "assert-not-exists":
			_, err := page.WaitForSelector(step.Target, playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(3000), State: playwright.WaitForSelectorStateDetached})
			if err != nil {
				logger.warn(scope, "assert-not-exists failed", map[string]any{"error": err.Error(), "target": step.Target})
			} else {
				logger.info(scope, "assert-not-exists ok", map[string]any{"target": step.Target})
			}
		case "assert-attr":
			val, err := page.GetAttribute(step.Target, step.Attr)
			if err != nil {
				logger.warn(scope, "assert-attr failed", map[string]any{"error": err.Error(), "target": step.Target, "attr": step.Attr})
				continue
			}
			if val != step.Value {
				logger.warn(scope, "assert-attr mismatch", map[string]any{"target": step.Target, "attr": step.Attr, "expected": step.Value, "got": val})
			} else {
				logger.info(scope, "assert-attr ok", map[string]any{"target": step.Target, "attr": step.Attr, "value": val})
			}
		default:
			logger.warn(scope, "unknown action", map[string]any{"action": step.Action})
		}
	}
}

func fetchScript(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("bad status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp("", "userscript-url-*.user.js")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

func fetchScriptFromGit(repo, filePath string) (string, error) {
	if repo == "" || filePath == "" {
		return "", errors.New("git repo and file path required")
	}
	dir, err := os.MkdirTemp("", "userscript-git-")
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "clone", "--depth", "1", repo, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone: %v: %s", err, string(out))
	}
	target := filepath.Join(dir, filePath)
	data, err := os.ReadFile(target)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp("", "userscript-git-*.user.js")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}
