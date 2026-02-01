# Containers (macOS priority, Linux friendly)

## Prereqs
- Podman (recommended) or Docker
- Go 1.25.6+ if building locally
- macOS: optional apple/container runtime supported (no Windows)

## Build
```bash
podman build -f Containerfile -t userscript-lab .
```

## Run (API + UI served together)
```bash
podman run --rm -p 8787:8787 \
  -v $(pwd)/runs:/app/runs \
  -v $(pwd)/extensions:/app/extensions \
  userscript-lab
```
- Open `http://localhost:8787/ui/` for the UI.
- Artifacts/HAR/trace go to `/app/runs`; mount `runs/` to persist.

## Extensions (bring-your-own)
- Tampermonkey MV3 CRX: https://clients2.google.com/service/update2/crx?response=redirect&prodversion=137&x=id%3Ddhdgffkkebhmkfjojejmpbldmpobfkfo%26installsource%3Dondemand%26uc
- Violentmonkey (Firefox): https://violentmonkey.github.io/get-it/
- Unzip locally to `extensions/tampermonkey-mv3` or `extensions/violentmonkey-firefox` before running, or upload via the UI Extensions panel (saves to `/app/extensions`).

## AlmaLinux note
- Verified with podman on AlmaLinux; same build/run commands and volume mounts.

## Image details
- Distroless base, no shell inside; rebuild for changes.
- Entrypoint runs `lab serve --port 8787`, exposing `/v1/runs` API, artifacts under `/runs/`, and static UI under `/ui/`.
