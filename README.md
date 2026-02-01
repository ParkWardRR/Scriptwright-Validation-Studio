# Scriptwright Validation Studio

**A tool for testing userscripts with real browsers and browser extensions.**

![Status](https://img.shields.io/badge/status-alpha%20v0.6-orange)
![Go](https://img.shields.io/badge/go-1.25-blue?logo=go)
![Playwright](https://img.shields.io/badge/playwright--go-0.5200-green?logo=playwright)
![License](https://img.shields.io/badge/license-Blue%20Oak%201.0.0-purple)

---

## What is this?

This tool lets you:
1. **Load a userscript** (like one from GreasyFork)
2. **Test it in a real browser** (Chromium via Playwright)
3. **Get proof it works** (screenshots, videos, logs)

Think of it as automated QA for userscripts.

---

## Quick Start

### Option 1: Run Locally

```bash
# Install dependencies (macOS)
brew install ffmpeg webp podman

# Build the container
podman build -f Containerfile -t userscript-lab .

# Run it
podman run --rm -p 8787:8787 -v $(pwd)/runs:/app/runs userscript-lab

# Open the UI
open http://localhost:8787/ui/
```

### Option 2: Deploy to Server

```bash
# Deploy to remote server
./deploy.sh

# Access it
# Direct: http://scriptwright:8787/ui/
# Or via SSH tunnel: ssh -i ~/.ssh/scriptwright -L 8787:localhost:8787 alfa@scriptwright
```

---

## How It Works

### Architecture

```
┌──────────────┐
│   Web UI     │  ← You interact here
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Go Server   │  ← API server (port 8787)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Playwright  │  ← Launches Chromium, runs scripts
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Artifacts   │  ← Saves screenshots, videos, logs
└──────────────┘
```

### What Gets Generated

When you run a test, you get:
- **Screenshot** (PNG)
- **Video** (WebP format)
- **Logs** (structured JSON)
- **Manifest** (JSON summary of the run)
- *Optional:* HAR file (network traffic), Trace file (debugging)

All saved in: `runs/{run-id}/artifacts/`

---

## Usage

### Web UI (Easiest)

1. Open http://localhost:8787/ui/
2. Fill in:
   - **Target URL:** The website to test on
   - **Script:** Path, URL, or Git repo of your userscript
   - **Engine:** How to run it (Init Script or Tampermonkey)
3. Click **Run via API**
4. View results (screenshot, logs, artifacts)

### CLI (Advanced)

```bash
# Run a single test
go run ./cmd/lab run \
  --url https://en.wikipedia.org/wiki/Tampermonkey \
  --script scripts/wikipedia-dark.user.js \
  --headless=true

# Start the API server
go run ./cmd/lab serve --port 8787

# List previous runs
go run ./cmd/lab list
```

### API (Programmatic)

```bash
# Health check
curl http://localhost:8787/health

# Create a run
curl -X POST http://localhost:8787/v1/runs \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "script": "path/to/script.user.js",
    "engine": "Tampermonkey (init-script)",
    "headless": true
  }'

# Get run results
curl http://localhost:8787/v1/runs/{run-id}
```

---

## Features

### ✅ What Works

- **Script Loading:** From file, URL, or Git repo
- **Browser Automation:** Chromium via Playwright-Go
- **Screenshot Capture:** Full-page PNG
- **Video Recording:** WebM → WebP conversion
- **Logging:** Structured NDJSON logs
- **Visual Regression:** Hash-based baseline comparison with pixel diff
- **Network Assertions:** Check for blocked hosts, status codes
- **Flow Testing:** Click, fill, wait, assert (DOM actions)
- **HAR Recording:** Capture network traffic
- **Trace Recording:** Playwright trace files
- **Web UI:** Settings, console, artifact preview
- **Deployment:** Container + systemd service

### 🟡 Partially Working

- **Tampermonkey Loading:** Attempts to load extension (requires manual setup)
- **Violentmonkey:** Not automated
- **Trace Capture:** Some timing bugs
- **HAR Replay:** Basic support, minimal error handling

### ❌ Not Built Yet

- React/TypeScript UI (current: vanilla JS)
- Project/suite persistence (current: single runs)
- Real-time log streaming (current: static files)
- Embedded trace/HAR viewers
- CI/CD integration
- Retry logic
- Performance metrics

---

## Project Structure

```
philadelphia/
├── cmd/
│   ├── lab/         # Main CLI (run, serve, list)
│   ├── demo/        # Demo runner (generates artifacts)
│   └── capture_ui/  # UI screenshot utility
├── internal/
│   ├── runner/      # Core Playwright orchestration
│   └── userscript/  # Userscript metadata parser
├── webui/           # Web UI (HTML/CSS/JS)
│   ├── index.html
│   ├── app.js
│   └── style.css
├── scripts/         # Example userscripts
├── runs/            # Generated test runs (gitignored)
├── Containerfile    # Container build
├── deploy.sh        # Remote deployment script
└── userscript-lab.service  # Systemd service
```

---

## Configuration

### Environment Variables

```bash
# Extension directory
export USERSCRIPT_ENGINE_EXT_DIR=/path/to/extensions

# Visual regression baseline
export BASELINE_DIR=/path/to/baselines

# Blocked hosts (comma-separated)
export BLOCKED_HOSTS=ads.example.com,tracker.com
```

### CLI Flags

```bash
# Run command
--url          Target URL
--script       Script path/URL/git repo
--engine       Engine type (default: "Tampermonkey (init-script)")
--ext          Extension directory
--headless     Headless mode (default: true)
--trace        Capture trace (default: false)
--har          Capture HAR (default: false)
--baseline     Baseline directory for visual diff
--steps        JSON flow steps

# Serve command
--port         Port to listen on (default: 8787)
```

---

## Deployment

### Server Requirements

- OS: AlmaLinux / RHEL / Debian / Ubuntu
- Podman or Docker
- Firewall: Open port 8787
- Optional: Systemd for service management

### Current Production Deployment

**Server:** `scriptwright.alpina` (alfa@scriptwright)
**URL:** http://scriptwright:8787/ui/
**Service:** systemd (userscript-lab.service)
**Status:** ✅ Running

```bash
# Check status
ssh -i ~/.ssh/scriptwright alfa@scriptwright "sudo systemctl status userscript-lab"

# View logs
ssh -i ~/.ssh/scriptwright alfa@scriptwright "sudo journalctl -u userscript-lab -f"

# Restart
ssh -i ~/.ssh/scriptwright alfa@scriptwright "sudo systemctl restart userscript-lab"
```

---

## Development

### Run Tests

```bash
go test ./...
```

### Build Binary

```bash
go build -o lab ./cmd/lab
./lab serve
```

### Build Container

```bash
podman build -f Containerfile -t userscript-lab .
podman run --rm -p 8787:8787 userscript-lab
```

---

## Examples

### Test a Wikipedia Dark Mode Script

```bash
go run ./cmd/lab run \
  --url https://en.wikipedia.org/wiki/Tampermonkey \
  --script scripts/wikipedia-dark.user.js \
  --headless=true
```

### Test with Visual Regression

```bash
# First run (creates baseline)
go run ./cmd/lab run --url https://example.com --script test.user.js --baseline ./baselines

# Second run (compares to baseline)
go run ./cmd/lab run --url https://example.com --script test.user.js --baseline ./baselines
# Output: visual_diff_img if pixels changed
```

### Test with Flow Steps

```bash
go run ./cmd/lab run \
  --url https://example.com \
  --script test.user.js \
  --steps '[
    {"action":"click","target":"text=Accept Cookies"},
    {"action":"wait","target":"1000"},
    {"action":"assert-text","target":"Welcome"}
  ]'
```

---

## Troubleshooting

### "podman: command not found"

Install podman:
```bash
# macOS
brew install podman

# AlmaLinux/RHEL
sudo dnf install -y podman
```

### "Port 8787 already in use"

Change the port:
```bash
go run ./cmd/lab serve --port 9000
```

### "Playwright browser not found"

The container includes Playwright. If running locally without container:
```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install
```

### "Extension not loading"

Extensions require manual setup. Download:
- Tampermonkey MV3: https://clients2.google.com/service/update2/crx?response=redirect&prodversion=137&x=id%3Ddhdgffkkebhmkfjojejmpbldmpobfkfo%26installsource%3Dondemand%26uc
- Violentmonkey: https://violentmonkey.github.io/get-it/

Unzip to `extensions/` directory.

---

## What's Next?

See [NEXT.md](./NEXT.md) for the roadmap and next steps.

See [container.md](./container.md) for deployment details.

---

## License

Blue Oak Model License 1.0.0 — See [LICENSE](./LICENSE)

---

## Contributing

This is an alpha project. Contributions welcome but expect breaking changes.

**Current Status:** 45% complete
- ✅ Core runner works
- ✅ Basic UI works
- 🟡 Extension loading needs work
- ❌ Advanced features (React UI, persistence, CI) not built

See [roadmap.md](./roadmap.md) for full feature status.
