# Running with apple/container + podman (macOS ARM/Intel)

## Prereqs
- macOS with Apple’s container runtime installed: https://github.com/apple/container
- Podman (recommended) or Docker
- Go 1.25.6+ if building locally

## Build the image
```bash
podman build -f Containerfile -t userscript-lab .
```

## Run the API + UI
```bash
podman run --rm -p 8787:8787 -v $(pwd)/runs:/app/runs userscript-lab
```
- Open `webui/index.html` locally (or serve via `python -m http.server 8000`) to use the UI against the container API at `http://localhost:8787`.

## Extensions (TM/VM) on Apple container
- Extensions are **not bundled**. Download locally and mount:
  - Tampermonkey MV3 CRX: https://clients2.google.com/service/update2/crx?response=redirect&prodversion=137&x=id%3Ddhdgffkkebhmkfjojejmpbldmpobfkfo%26installsource%3Dondemand%26uc
  - Violentmonkey (Firefox): https://violentmonkey.github.io/get-it/
- Unzip to `extensions/tampermonkey-mv3` or `extensions/violentmonkey-firefox`.
- Mount into the container:
  ```bash
  podman run --rm -p 8787:8787 \
    -v $(pwd)/runs:/app/runs \
    -v $(pwd)/extensions:/app/extensions \
    userscript-lab
  ```
- In the UI Extensions panel, you can also upload CRX/XPI; the backend saves to `/app/extensions`.

## Notes
- The container entrypoint serves only the API (`lab serve`). The UI remains static; open it from your host.
- Trace/HAR and screenshots are written to `runs/` (mount to persist).
- Apple/container works with distroless base; no shell inside the image. Build/changes must happen outside then rebuild.
