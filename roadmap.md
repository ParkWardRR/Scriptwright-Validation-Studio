# roadmap.md — Development roadmap

## Milestones (checklist)
- [ ] M0 — Spike (goal: extension + userscript automation feasibility)  
  - [x] Demo run produced with Playwright persistent context and userscript execution (init-script injection) plus artifacts (PNG, WebP, manifest).  
  - [~] MV3 extension load + deterministic TM/VM install path (API supports `--ext` dir; extension asset still needed/validated).
- [ ] M1 — MVP runner (deterministic runs + artifacts)  
  - [x] `lab run` CLI with persistent profile mgmt.  
  - [~] Script ingestion (file path supported; paste/git to follow).  
  - [x] Basic DOM assertion + NDJSON logs.
- [ ] M2 — UI v1 (usable pro workflow)  
  - [x] Prototype web UI shell with settings, embedded browser pane, console stream, and artifact preview.  
  - [x] Wire UI to runner API for live runs and artifact retrieval (falls back to simulation when API offline).
- [ ] M3 — “Testing to the max” (full checklist categories)  
  - [~] Visual regression + network assertions + HAR record toggle (hash-based baseline + blocked-host/status checks + HAR capture stub).  
  - [ ] Flow editor (step list) driving Playwright.
- [ ] M4 — Hardening (reduce flake + better debugging)  
  - [ ] Trace-first workflow; retry policy; selector diagnostics; redaction; export bundles.
- [ ] M5 — CI & sharing (team-scale)  
  - [ ] Headless/headed strategy; GitHub Actions template; artifact upload; JSON report format.

## Workstreams (status)
- [ ] Extension management — must follow Playwright extension constraints (persistent context; bundled Chromium). [page:0]
- [ ] Script ingestion/versioning — paste/file in progress; git URL later.
- [ ] Runner observability — traces/NDJSON/screenshots planned; sample manifest + video now exist. [web:51][web:52]
- [x] UI debug console — prototype shell with console stream + artifact preview added.
- [ ] Validation framework — design schema + assertion types to come.

## Definition of Done (per milestone)
- M0: One-click demo with extension load + URL run + artifacts saved + constraints documented.
- M1: CLI stable; results JSON; logs + screenshots; deterministic workspace layout.
- M2: UI can create suite + run + inspect artifacts without touching filesystem.
- M3: Checklist supports DOM + visual + network + flow in one run; baselines stored per suite.
- M4: Exportable run bundle; redaction; retry-on-fail; clear error taxonomy.
- M5: CI template + docs; sample repo; reproducible pinned versions.

## Done since last update
- Captured Playwright-Go demo artifacts (PNG, WebP, run manifest) for Wikipedia Dark/Light userscript.
- Added userscript metadata parser with unit tests.
- Published README with hero WebP and quickstart.
- Dropped Blue Oak Model License 1.0.0 into repo.
- Built a prototype web UI shell (settings + browser pane + console + artifact preview).

## Known risks / decision points
| Risk | Impact | Mitigation |
|---|---|---|
| VM on Chromium (MV2 deprecation) | “VM option” may not work on modern Chromium | Provide clear matrix + recommend TM for Chromium; keep VM as “best effort” or route VM runs to Firefox profiles. [web:50] |
| TM MV3 feature gaps | Some userscripts relying on unsupported APIs may fail | Detect GM API usage, warn before run, link to TM MV3 limitations. [web:45][web:21] |
| Extension UI automation brittleness | Flaky script install flows | Pin extension versions; treat selector updates as maintenance; keep M0 focused on a stable path |

## Research sources (official-first)
- Playwright “Chrome extensions” documentation (persistent context, args, MV3 service worker + extensionId). [page:0]
- Playwright BrowserContext API (events, routing/HAR replay caveats w/ service workers, storageState). [page:1]
- Playwright Trace Viewer / Tracing docs (trace artifacts + debugging workflow). [web:51][web:52]
- Playwright-Go installation/version pinning guidance. [web:11][web:9]
- Tampermonkey documentation/changelog notes about MV3 and MV3 limitations. [web:21][web:45]
- Violentmonkey official site note about Chrome/MV2 status. [web:50]
- Chromium extensions discussion on MV3 remotely-hosted code restriction. [web:41]
