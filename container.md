# Containers (macOS + Linux)

## Prereqs
- Podman (recommended) or Docker
- Go 1.25.6+ if building locally
- macOS: optional apple/container runtime (https://github.com/apple/container) for native containers

## Build (macOS or Linux)
```bash
podman build -f Containerfile -t userscript-lab .
```

## Run API + UI
```bash
podman run --rm -p 8787:8787 -v $(pwd)/runs:/app/runs userscript-lab
```
- Open `webui/index.html` locally (or serve via `python -m http.server 8000`) to use the UI against `http://localhost:8787`.

## Extensions (not bundled)
- Tampermonkey MV3 CRX: https://clients2.google.com/service/update2/crx?response=redirect&prodversion=137&x=id%3Ddhdgffkkebhmkfjojejmpbldmpobfkfo%26installsource%3Dondemand%26uc
- Violentmonkey (Firefox): https://violentmonkey.github.io/get-it/
- Unzip to `extensions/tampermonkey-mv3` or `extensions/violentmonkey-firefox`, then mount:
```bash
podman run --rm -p 8787:8787 \
  -v $(pwd)/runs:/app/runs \
  -v $(pwd)/extensions:/app/extensions \
  userscript-lab
```
- Or upload via the Extensions panel in the web UI; backend saves to `/app/extensions`.

## AlmaLinux note
- Works out of the box on AlmaLinux (tested with podman). Use the same build/run commands as above; mount `runs/` and `extensions/` for persistence.

## Notes
- Entry point serves the API only (`lab serve`). UI is static from host.
- Trace/HAR/screenshots land in `runs/` (mount to persist).
- Distroless base: no shell inside; rebuild for changes.
