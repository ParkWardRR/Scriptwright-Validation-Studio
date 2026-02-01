package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"philadelphia/internal/userscript"

	"github.com/playwright-community/playwright-go"
)

const (
	wikiURL          = "https://en.wikipedia.org/wiki/Tampermonkey"
	scriptFilename   = "wikipedia-dark.user.js"
	artifactsDirName = "artifacts"
	videoOutputName  = "wikipedia-dark.webp"
)

type runManifest struct {
	StartedAt     time.Time       `json:"started_at"`
	FinishedAt    time.Time       `json:"finished_at"`
	TargetURL     string          `json:"target_url"`
	Screenshot    string          `json:"screenshot"`
	VideoWebP     string          `json:"video_webp"`
	ScriptMeta    userscript.Meta `json:"script_meta"`
	ProfileFolder string          `json:"profile_folder"`
}

func main() {
	logger := log.New(os.Stdout, "[demo] ", log.LstdFlags|log.Lmicroseconds)

	artifactsDir := filepath.Join(artifactsDirName)
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		logger.Fatalf("create artifacts dir: %v", err)
	}

	scriptPath := filepath.Join("scripts", scriptFilename)
	meta, err := userscript.Parse(scriptPath)
	if err != nil {
		logger.Fatalf("parse userscript: %v", err)
	}
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		logger.Fatalf("read script: %v", err)
	}

	if err := ensurePlaywright(logger); err != nil {
		logger.Fatalf("install Playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		logger.Fatalf("start Playwright: %v", err)
	}
	defer pw.Stop()

	profileDir := filepath.Join(os.TempDir(), "philadelphia-profile")
	_ = os.RemoveAll(profileDir) // clean up stale state; errors non-fatal

	ctx, err := pw.Chromium.LaunchPersistentContext(profileDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(true),
		RecordVideo: &playwright.RecordVideo{
			Dir:  filepath.Join(artifactsDir, "video"),
			Size: &playwright.Size{Width: 1280, Height: 720},
		},
		Args: []string{"--disable-dev-shm-usage"},
	})
	if err != nil {
		logger.Fatalf("launch persistent context: %v", err)
	}

	page, err := ctx.NewPage()
	if err != nil {
		logger.Fatalf("new page: %v", err)
	}

	start := time.Now()
	logger.Printf("Navigating to %s ...", wikiURL)
	if _, err := page.Goto(wikiURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30_000),
	}); err != nil {
		logger.Fatalf("navigate: %v", err)
	}

	if _, err := page.Evaluate(string(scriptContent)); err != nil {
		logger.Fatalf("execute userscript: %v", err)
	}

	if _, err := page.WaitForSelector("text=Toggle Dark Mode", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10_000),
	}); err != nil {
		logger.Fatalf("wait for toggle button: %v", err)
	}
	logger.Println("Toggling dark mode")
	if err := page.Click("text=Toggle Dark Mode"); err != nil {
		logger.Fatalf("click toggle: %v", err)
	}

	// Let the UI settle before capturing.
	page.WaitForTimeout(1500)

	screenshotPath := filepath.Join(artifactsDir, "wikipedia-dark.png")
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	}); err != nil {
		logger.Fatalf("screenshot: %v", err)
	}

	video := page.Video()
	if err := page.Close(); err != nil {
		logger.Fatalf("close page: %v", err)
	}
	if err := ctx.Close(); err != nil {
		logger.Fatalf("close context: %v", err)
	}

	var (
		videoPath string
		webpPath  = filepath.Join(artifactsDir, videoOutputName)
	)
	if video != nil {
		videoPath, err = video.Path()
		if err != nil {
			logger.Printf("video path error (continuing): %v", err)
		}
	}
	if videoPath != "" {
		if err := convertToWebP(videoPath, webpPath); err != nil {
			logger.Printf("convert video to webp failed: %v", err)
		} else {
			logger.Printf("Saved webp video to %s", webpPath)
		}
	} else {
		logger.Println("No video path returned; webp preview skipped.")
	}

	manifest := runManifest{
		StartedAt:     start,
		FinishedAt:    time.Now(),
		TargetURL:     wikiURL,
		Screenshot:    screenshotPath,
		VideoWebP:     webpPath,
		ScriptMeta:    meta,
		ProfileFolder: profileDir,
	}
	manifestPath := filepath.Join(artifactsDir, "run.json")
	if err := writeManifest(manifestPath, manifest); err != nil {
		logger.Printf("write manifest failed: %v", err)
	} else {
		logger.Printf("Run manifest saved to %s", manifestPath)
	}

	logger.Println("Demo complete. Open README.md to see the embedded assets.")
}

func ensurePlaywright(logger *log.Logger) error {
	logger.Println("Ensuring Playwright browsers are installed (chromium)...")
	return playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}})
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
	cmd := exec.Command("ffmpeg", "-y", "-i", input, "-vcodec", "libwebp", "-filter:v", "fps=15,scale=1280:-1:flags=lanczos", "-loop", "0", "-an", "-vsync", "0", output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeManifest(path string, manifest runManifest) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(manifest)
}
