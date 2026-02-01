package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/playwright-community/playwright-go"
)

func main() {
	artifactsDir := "artifacts"
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		log.Fatalf("create artifacts dir: %v", err)
	}

	htmlPath, err := filepath.Abs(filepath.Join("webui", "index.html"))
	if err != nil {
		log.Fatalf("resolve html path: %v", err)
	}
	target := "file://" + htmlPath

	if err := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}}); err != nil {
		log.Fatalf("install playwright: %v", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("start playwright: %v", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("launch browser: %v", err)
	}
	defer browser.Close()

	page, err := browser.NewPage(playwright.BrowserNewPageOptions{
		Viewport: &playwright.Size{Width: 1400, Height: 900},
	})
	if err != nil {
		log.Fatalf("new page: %v", err)
	}

	if _, err := page.Goto(target, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		log.Fatalf("goto: %v", err)
	}

	page.WaitForTimeout(1200)

	screenshotPath := filepath.Join(artifactsDir, "webui.png")
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(screenshotPath),
		FullPage: playwright.Bool(true),
	}); err != nil {
		log.Fatalf("screenshot: %v", err)
	}

	log.Printf("Captured web UI screenshot at %s", screenshotPath)
}
