# roadmap.md — Development roadmap

## Milestones (table)
| Milestone | Goal | Deliverables | Timebox |
|---|---|---|---|
| M0 — Spike | Prove extension + userscript automation feasibility | Load MV3 extension via persistent context; confirm stable way to install/enable a script; produce a demo run w/ artifacts | 1 week |
| M1 — MVP runner | Deterministic runs + artifacts | Go runner with `lab run`, persistent profile mgmt, script ingestion (paste + file), basic DOM assertions, NDJSON logs | 2–3 weeks |
| M2 — UI v1 | Usable pro workflow | React/TS app: project → suite → run dashboard; live log console; artifact viewer; basic settings (TM/VM) | 2–3 weeks |
| M3 — “Testing to the max” | Full checklist categories | Visual regression workflow, network assertion module, HAR record/replay toggle, flow editor (step list) | 3–4 weeks |
| M4 — Hardening | Reduce flake + better debugging | Trace-first workflow, run retry policy, timeouts defaults, selector diagnostics, redaction, export bundles | 2–3 weeks |
| M5 — CI & sharing | Team-scale usage | Headless/headed strategy finalized, GitHub Actions template, artifact upload, JSON report format | 2 weeks |

## Workstreams (what to build in parallel)
| Workstream | Why | Notes |
|---|---|---|
| Extension management | Core technical risk | Must follow Playwright extension constraints (persistent context; bundled Chromium). [page:0] |
| Script ingestion/versioning | Reproducibility | Support paste/file first; add git URL next |
| Runner observability | Makes it “pro” | Traces + NDJSON + screenshots early; Trace Viewer integration is key. [web:51][web:52] |
| UI debug console | Your differentiator | Unified log stream with filtering + correlation |
| Validation framework | “All of the above” checklist | Design schema once, then add assertion types incrementally |

## Definition of Done per milestone
| Milestone | DoD |
|---|---|
| M0 | One-click demo: launch context with extension + run on a URL; artifacts saved; documented constraints | 
| M1 | CLI stable; results JSON; logs and screenshots; deterministic workspace layout |
| M2 | UI can create suite + run + inspect artifacts without touching filesystem |
| M3 | Checklist supports DOM+visual+network+flow in one run; baselines stored per suite |
| M4 | Exportable run bundle; redaction; retry-on-fail; clear error taxonomy |
| M5 | CI template + docs; sample repo; reproducible pinned versions |

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
