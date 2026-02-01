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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"philadelphia/internal/userscript"

	"github.com/playwright-community/playwright-go"
)

// Options configure a run.
type Options struct {
	TargetURL    string
	ScriptPath   string
	ExtensionDir string // optional: path to unpacked MV3 extension (e.g., Tampermonkey)
	Engine       string // display only for now
	Headless     bool
	ProfileDir   string // optional persistent profile location
	CaptureTrace bool
	CaptureHAR   bool
	ReplayHAR    string   // optional path to HAR for replay
	BaselineDir  string   // for visual regression hashes
	BlockedHosts []string // basic network assertion
	Workspace    string   // base path; defaults to cwd
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
	RunID         string          `json:"run_id"`
	StartedAt     time.Time       `json:"started_at"`
	FinishedAt    time.Time       `json:"finished_at"`
	TargetURL     string          `json:"target_url"`
	Screenshot    string          `json:"screenshot"`
	VideoWebM     string          `json:"video_webm,omitempty"`
	VideoWebP     string          `json:"video_webp,omitempty"`
	TraceZip      string          `json:"trace_zip,omitempty"`
	HAR           string          `json:"har,omitempty"`
	ReplayHAR     string          `json:"replay_har,omitempty"`
	ScriptMeta    userscript.Meta `json:"script_meta"`
	ProfileFolder string          `json:"profile_folder"`
	Engine        string          `json:"engine"`
	ExtensionDir  string          `json:"extension_dir,omitempty"`
	LogPath       string          `json:"log_path"`
	VisualHash    string          `json:"visual_hash,omitempty"`
	VisualDiff    bool            `json:"visual_diff,omitempty"`
	VisualDiffImg string          `json:"visual_diff_img,omitempty"`
	NetworkIssues []string        `json:"network_issues,omitempty"`
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
	if err := page.AddInitScript(playwright.Script{Content: playwright.String(string(scriptContent))}); err != nil {
		logger.warn("runner", "init script injection failed; continuing", map[string]any{"error": err.Error()})
	}

	start := time.Now()
	logger.info("browser", "navigating", map[string]any{"url": opts.TargetURL})
	if _, err := page.Goto(opts.TargetURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(40_000),
	}); err != nil {
		return Result{}, fmt.Errorf("navigate: %w", err)
	}

	// Minimal DOM assertion to satisfy M1 goals.
	if _, err := page.WaitForSelector("text=Toggle Dark Mode", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(8_000),
	}); err != nil {
		logger.warn("assert", "toggle button not found", map[string]any{"error": err.Error()})
	} else {
		logger.info("assert", "toggle button present", nil)
	}

	// Toggle to show behavior difference.
	if err := page.Click("text=Toggle Dark Mode"); err != nil {
		logger.warn("action", "click toggle failed", map[string]any{"error": err.Error()})
	} else {
		logger.info("action", "toggled dark mode", nil)
	}

	page.WaitForTimeout(1200)

	screenshotPath := filepath.Join(artifactsDir, "screenshot.png")
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	}); err != nil {
		logger.warn("artifact", "screenshot failed", map[string]any{"error": err.Error()})
	}

	visualHash, visualDiff, visualDiffImg := computeVisualHash(screenshotPath, opts, artifactsDir, logger)

	video := page.Video()
	if err := page.Close(); err != nil {
		logger.warn("runner", "close page", map[string]any{"error": err.Error()})
	}

	if opts.CaptureTrace {
		// TODO: integrate playwright tracing when stabilizing
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
		RunID:         runID,
		StartedAt:     start,
		FinishedAt:    time.Now(),
		TargetURL:     opts.TargetURL,
		Screenshot:    filepath.Base(screenshotPath),
		VisualHash:    visualHash,
		VisualDiff:    visualDiff,
		VisualDiffImg: visualDiffImg,
		VideoWebM:     filepath.Base(videoPath),
		VideoWebP:     filepath.Base(webpPath),
		TraceZip:      "",
		HAR:           harNameIfExists(artifactsDir),
		ReplayHAR:     opts.ReplayHAR,
		ScriptMeta:    scriptMeta,
		ProfileFolder: profileDir,
		Engine:        opts.Engine,
		ExtensionDir:  opts.ExtensionDir,
		LogPath:       logPath,
		NetworkIssues: summarizeNetwork(responses, opts.BlockedHosts, logger),
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
func computeVisualHash(screenshotPath string, opts Options, artifactsDir string, logger *ndjsonLogger) (hash string, diff bool, diffImg string) {
	data, err := os.ReadFile(screenshotPath)
	if err != nil || len(data) == 0 {
		return "", false, ""
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	hash = sum
	if opts.BaselineDir == "" {
		return hash, false, ""
	}
	if err := os.MkdirAll(opts.BaselineDir, 0o755); err != nil {
		logger.warn("visual", "baseline dir create failed", map[string]any{"error": err.Error()})
		return hash, false, ""
	}
	basePath := filepath.Join(opts.BaselineDir, "screenshot.png")
	if _, err := os.Stat(basePath); err != nil {
		_ = os.WriteFile(basePath, data, 0o644)
		logger.info("visual", "baseline created", map[string]any{"path": basePath})
		return hash, false, ""
	}
	baseData, err := os.ReadFile(basePath)
	if err != nil {
		logger.warn("visual", "baseline read failed", map[string]any{"error": err.Error()})
		return hash, false, ""
	}
	baseHash := fmt.Sprintf("%x", sha256.Sum256(baseData))
	if baseHash != hash {
		logger.warn("visual", "screenshot hash mismatch vs baseline", map[string]any{"baseline": baseHash, "current": hash})
		diffImg = computeDiffImage(baseData, data, artifactsDir, logger)
		return hash, true, diffImg
	}
	logger.info("visual", "screenshot matches baseline", nil)
	return hash, false, ""
}

func harNameIfExists(artifactsDir string) string {
	har := filepath.Join(artifactsDir, "network.har")
	if _, err := os.Stat(har); err == nil {
		return filepath.Base(har)
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

// computeDiffImage generates a simple diff heatmap (red overlay) when sizes match.
func computeDiffImage(basePNG, currentPNG []byte, artifactsDir string, logger *ndjsonLogger) string {
	baseImg, err := png.Decode(bytes.NewReader(basePNG))
	if err != nil {
		logger.warn("visual", "decode baseline failed", map[string]any{"error": err.Error()})
		return ""
	}
	currImg, err := png.Decode(bytes.NewReader(currentPNG))
	if err != nil {
		logger.warn("visual", "decode current failed", map[string]any{"error": err.Error()})
		return ""
	}
	if baseImg.Bounds() != currImg.Bounds() {
		logger.warn("visual", "baseline/current size mismatch", nil)
		return ""
	}
	bounds := baseImg.Bounds()
	diff := image.NewRGBA(bounds)
	var changed int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			br, bg, bb, _ := baseImg.At(x, y).RGBA()
			cr, cg, cb, _ := currImg.At(x, y).RGBA()
			if br != cr || bg != cg || bb != cb {
				changed++
				diff.Set(x, y, color.RGBA{R: 255, A: 180})
			} else {
				diff.Set(x, y, color.RGBA{A: 0})
			}
		}
	}
	if changed == 0 {
		return ""
	}
	out := filepath.Join(artifactsDir, "visual-diff.png")
	f, err := os.Create(out)
	if err != nil {
		logger.warn("visual", "write diff failed", map[string]any{"error": err.Error()})
		return ""
	}
	defer f.Close()
	if err := png.Encode(f, diff); err != nil {
		logger.warn("visual", "encode diff failed", map[string]any{"error": err.Error()})
		return ""
	}
	logger.warn("visual", "diff image generated", map[string]any{"path": out, "pixels_changed": changed})
	return filepath.Base(out)
}
