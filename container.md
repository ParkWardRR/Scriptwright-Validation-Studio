# Container Guide — Deployment & Running

**Quick reference for deploying and running the userscript lab in containers.**

---

## Prerequisites

### macOS
```bash
brew install podman ffmpeg webp
```

### Linux (AlmaLinux/RHEL/Fedora)
```bash
sudo dnf install -y podman
```

### Linux (Debian/Ubuntu)
```bash
sudo apt install -y podman
```

---

## Quick Start

### 1. Build the Container

```bash
podman build -f Containerfile -t userscript-lab .
```

**What this does:**
- Builds a Go binary from source
- Creates a minimal distroless container
- Includes Playwright + Chromium
- Exposes port 8787

### 2. Run Locally

```bash
podman run --rm -p 8787:8787 \
  -v $(pwd)/runs:/app/runs \
  -v $(pwd)/extensions:/app/extensions \
  userscript-lab
```

**What this does:**
- Maps port 8787 (host) → 8787 (container)
- Mounts `runs/` directory for artifacts
- Mounts `extensions/` directory for browser extensions
- Auto-starts the API server

### 3. Open the UI

```bash
open http://localhost:8787/ui/
```

Or visit: http://localhost:8787/ui/

---

## Deployment to Remote Server

### Automated Deployment

```bash
# Deploy to remote server (configured in deploy.sh)
./deploy.sh
```

**What this does:**
1. Copies source code to remote server via rsync
2. Builds container on remote server
3. Stops old container (if running)
4. Starts new container with systemd
5. Configures firewall (port 8787)

**Default target:** `alfa@scriptwright` (edit `deploy.sh` to change)

### Manual Deployment

```bash
# SSH into server
ssh user@server

# Clone repo
git clone <repo-url> /path/to/app
cd /path/to/app

# Build container
podman build -f Containerfile -t userscript-lab .

# Run container
podman run -d --name userscript-lab \
  --restart unless-stopped \
  -p 8787:8787 \
  -v /home/user/runs:/app/runs \
  -v /home/user/extensions:/app/extensions \
  userscript-lab
```

---

## Systemd Service (Recommended for Production)

### Install Service

```bash
# Copy service file to server
scp userscript-lab.service user@server:/tmp/

# SSH into server
ssh user@server

# Install service
sudo cp /tmp/userscript-lab.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable userscript-lab
sudo systemctl start userscript-lab
```

### Manage Service

```bash
# Check status
sudo systemctl status userscript-lab

# View logs
sudo journalctl -u userscript-lab -f

# Restart
sudo systemctl restart userscript-lab

# Stop
sudo systemctl stop userscript-lab

# Disable auto-start
sudo systemctl disable userscript-lab
```

---

## Accessing the Server

### Direct Access (if hostname resolves)

```bash
# Open in browser
open http://scriptwright:8787/ui/
```

### SSH Tunnel (if behind firewall)

```bash
# Create tunnel
ssh -i ~/.ssh/your-key -L 8787:localhost:8787 user@server

# Then open in browser
open http://localhost:8787/ui/
```

### Public Access (requires firewall configuration)

```bash
# Open firewall port
sudo firewall-cmd --permanent --add-port=8787/tcp
sudo firewall-cmd --reload

# Access via IP/hostname
http://<server-ip>:8787/ui/
```

---

## Extensions Setup

Extensions (Tampermonkey, Violentmonkey) must be downloaded separately due to licensing/size.

### Download Extensions

**Tampermonkey MV3 (Chromium):**
```bash
# Download CRX file
curl -L "https://clients2.google.com/service/update2/crx?response=redirect&prodversion=137&x=id%3Ddhdgffkkebhmkfjojejmpbldmpobfkfo%26installsource%3Dondemand%26uc" \
  -o tampermonkey.crx

# Unzip
unzip tampermonkey.crx -d extensions/tampermonkey-mv3
```

**Violentmonkey (Firefox):**
- Visit: https://violentmonkey.github.io/get-it/
- Download XPI file
- Unzip to `extensions/violentmonkey-firefox/`

### Upload via UI

1. Open http://localhost:8787/ui/
2. Go to Extensions panel
3. Click "Upload Extension"
4. Select CRX/XPI file

---

## Configuration

### Environment Variables

```bash
# Extension directory
export USERSCRIPT_ENGINE_EXT_DIR=/path/to/extensions

# Baseline directory for visual regression
export BASELINE_DIR=/path/to/baselines

# Blocked hosts (comma-separated)
export BLOCKED_HOSTS=ads.example.com,tracker.com
```

### Container Ports

| Port | Service | Description |
|------|---------|-------------|
| 8787 | HTTP API + UI | Main application |

### Volume Mounts

| Host Path | Container Path | Purpose |
|-----------|----------------|---------|
| `./runs` | `/app/runs` | Test run artifacts (screenshots, videos, logs) |
| `./extensions` | `/app/extensions` | Browser extensions (TM, VM) |

---

## Troubleshooting

### "Container won't start"

Check logs:
```bash
podman logs userscript-lab
```

### "Port 8787 already in use"

Find what's using it:
```bash
sudo lsof -i :8787
# or
sudo ss -tulpn | grep 8787
```

Kill the process or change the port:
```bash
podman run --rm -p 9000:8787 userscript-lab
```

### "Permission denied on /app/runs"

Fix permissions:
```bash
# On host
chmod 755 runs
sudo chown -R $(id -u):$(id -g) runs
```

### "Browser not found"

The container includes Playwright + Chromium. If this error appears, rebuild:
```bash
podman build --no-cache -f Containerfile -t userscript-lab .
```

### "Extension not loading"

Extensions must be manually downloaded and placed in `extensions/` directory. See Extensions Setup above.

---

## Production Checklist

- [ ] Firewall configured (port 8787 open)
- [ ] Systemd service installed and enabled
- [ ] Firewall configured (port 8787 only)
- [ ] SSH key-based auth (no passwords)
- [ ] Regular backups of `/home/user/runs`
- [ ] Monitoring (systemd status, logs)
- [ ] Optional: SSL/TLS reverse proxy (nginx, caddy)
- [ ] Optional: Authentication (BasicAuth, OAuth)

---

## Current Production Instance

**Server:** scriptwright.alpina
**User:** alfa@scriptwright
**URL:** http://scriptwright:8787/ui/
**Service:** systemd (userscript-lab.service)
**Status:** ✅ Running

**Stats:**
- Deployed: 2026-01-31
- Uptime: Check with `systemctl status userscript-lab`
- Disk usage: ~3GB (cleaned up, optimized)

---

## Container Details

### Image Specs

- **Base:** `gcr.io/distroless/base-debian12` (minimal, no shell)
- **Build:** Multi-stage (Go binary + static files only)
- **Size:** ~500MB (includes Chromium)
- **Port:** 8787
- **Entrypoint:** `/usr/local/bin/lab serve --port 8787`

### Security

- No shell in container (distroless)
- Runs as non-root
- Minimal attack surface
- No unnecessary packages

### Supported Platforms

- ✅ macOS (Apple Silicon + Intel)
- ✅ Linux (x86_64)
- ✅ AlmaLinux / RHEL / Fedora
- ✅ Debian / Ubuntu
- ✅ Proxmox (with podman)
- ❌ Windows (not tested)

---

## Advanced Usage

### Build with Custom Base

```bash
# Use different Go version
podman build --build-arg GO_VERSION=1.26 -f Containerfile -t userscript-lab .
```

### Run with Custom Port

```bash
# Change host port only
podman run --rm -p 9000:8787 userscript-lab

# Change container port (requires rebuild)
# Edit Containerfile: ENTRYPOINT ["lab", "serve", "--port", "9000"]
```

### Run with Resource Limits

```bash
podman run --rm -p 8787:8787 \
  --memory=2g \
  --cpus=2 \
  userscript-lab
```

### Run with Custom Config

```bash
podman run --rm -p 8787:8787 \
  -e BASELINE_DIR=/baselines \
  -e BLOCKED_HOSTS=ads.com,tracker.com \
  -v $(pwd)/baselines:/baselines \
  userscript-lab
```

---

## Docker (Alternative to Podman)

All `podman` commands work with `docker`:

```bash
# Build
docker build -f Containerfile -t userscript-lab .

# Run
docker run --rm -p 8787:8787 -v $(pwd)/runs:/app/runs userscript-lab

# List
docker ps

# Logs
docker logs userscript-lab

# Stop
docker stop userscript-lab
```

---

## Next Steps

- See [README.md](./README.md) for usage guide
- See [NEXT.md](./NEXT.md) for development roadmap
- See [roadmap.md](./roadmap.md) for feature status
