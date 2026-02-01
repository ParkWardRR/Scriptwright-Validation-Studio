package main

import (
	"log"
	"os"
	"path/filepath"

	"philadelphia/internal/runner"
)

const (
	wikiURL          = "https://en.wikipedia.org/wiki/Tampermonkey"
	scriptFilename   = "wikipedia-dark.user.js"
	artifactsDirName = "artifacts"
)

func main() {
	logger := log.New(os.Stdout, "[demo] ", log.LstdFlags|log.Lmicroseconds)

	artifactsDir := filepath.Join(artifactsDirName)
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		logger.Fatalf("create artifacts dir: %v", err)
	}

	scriptPath := filepath.Join("scripts", scriptFilename)
	result, err := runner.Run(runner.Options{
		TargetURL:    wikiURL,
		ScriptPath:   scriptPath,
		Engine:       "Tampermonkey (init-script injection)",
		Headless:     true,
		CaptureTrace: false,
		CaptureHAR:   false,
		Workspace:    ".",
	})
	if err != nil {
		logger.Fatalf("demo run failed: %v", err)
	}

	copyArtifact(result.Artifacts.Screenshot, filepath.Join(artifactsDir, "wikipedia-dark.png"), logger)
	if result.Artifacts.VideoWebP != "" {
		copyArtifact(result.Artifacts.VideoWebP, filepath.Join(artifactsDir, "wikipedia-dark.webp"), logger)
	}
	copyArtifact(filepath.Join(result.RunDir, "run.json"), filepath.Join(artifactsDir, "run.json"), logger)
	logger.Println("Demo complete. Open README.md to see the embedded assets.")
}

func copyArtifact(src, dst string, logger *log.Logger) {
	if src == "" {
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		logger.Printf("copy %s failed: %v", src, err)
		return
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		logger.Printf("write %s failed: %v", dst, err)
		return
	}
	logger.Printf("Wrote %s", dst)
}
