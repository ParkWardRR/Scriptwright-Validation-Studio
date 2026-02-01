# spec.md — Userscript Test Lab (Playwright-Go + TM/VM + React)

## 0) One-liner
A desktop-first app that lets you ingest userscripts (paste/file/git URL), install them into Tampermonkey or Violentmonkey, and run UX/UI functional tests via Playwright-Go with maximal observability (logs, traces, HAR, screenshots, videos) and a pro debugging console in the UI. [page:0][web:11][web:45]

## in the read me use lots of fancy formating and i love badges. Hella badges pls 

## 1) Product goals
### 1.1 Goals
- Run *real* userscript engines (Tampermonkey and Violentmonkey) as browser extensions, not “simulated injection,” to match real-world behavior. [page:0][web:45]
- Support Manifest V3 extension testing workflows, including extension service workers and stable extension loading. [page:0]
- Provide a user-facing, checklist-based validation flow that can combine DOM assertions, visual regression, network assertions (including HAR), and end-to-end behavioral flows. [page:1][web:51][web:60]
- Make troubleshooting easy with first-class logs, trace viewer integration, and an in-app debug console that can be attached to a specific run. [web:51][web:52]

### 1.2 Non-goals (v1)
- Not a general-purpose “browser extension test runner” for arbitrary extensions beyond userscript engines; we focus on TM/VM + userscripts. [page:0][web:45]
- Not a cloud SaaS (initially); local-first, with optional CI execution later. [page:0]

## 2) Reality constraints (must be explicit)
- Playwright’s official guidance: extensions only work in Chromium when launched with a **persistent context**, using `--disable-extensions-except` and `--load-extension`. [page:0]
- Playwright notes that Google Chrome and Microsoft Edge removed the command-line flags required to side-load extensions, so the default path is using the Chromium bundled with Playwright. [page:0]
- Manifest V3 extensions use a service worker instead of background pages, and Playwright shows deriving the `extensionId` from the service worker URL. [page:0]
- MV3 restricts remotely hosted code (“all code executed by the extension be present…”), which directly impacts how userscript managers can fetch/execute scripts. [web:41]
- Tampermonkey has switched to Manifest V3 (Chrome/Chromium) and has MV3-specific feature gaps (example: `GM_webRequest` “not (yet) supported in Manifest V3”). [web:45]
- Tampermonkey docs explicitly note some APIs are not available in MV3 versions 5.2+ (Chrome and derivatives). [web:21]
- Violentmonkey’s official site states it is no longer supported on Chrome due to its Manifest V2 architecture (MV3 rewrite may be considered later). [web:50]

**Implication:** “TM vs VM” must be a setting, but the product must clearly communicate that VM on Chromium may be blocked/unstable depending on the browser’s MV2 policy, while TM is expected to be the primary Chromium MV3 path. [web:45][web:50]

## 3) Target users & use cases
### 3.1 Primary use cases
| Use case | Description | Success criteria |
|---|---|---|
| Quick repro | Paste a userscript, run it on a URL, see whether it breaks page UX | Run completes with logs + artifacts, checklist shows pass/fail [web:51][web:52] |
| Regression suite | Keep a suite of scripts and re-run after script changes | Deterministic run config, comparable artifacts, diffable results |
| Cross-engine sanity | Run same script under TM and VM settings | Report highlights behavioral differences, with evidence artifacts |

### 3.2 Personas
| Persona | Needs |
|---|---|
| Script author | Fast iteration, clear failure causes, DOM + network evidence |
| Maintainer | CI-friendly, reproducible runs, artifact retention |
| Power user | Deep console, timeline/trace, ability to inspect requests/responses |

## 4) High-level architecture
### 4.1 Components
| Component | Tech | Responsibilities |
|---|---|---|
| Runner daemon | Go | Owns job queue, launches Playwright-Go, collects artifacts/logs, exposes local API |
| Browser automation | Playwright-Go | Launch persistent context, load extension, drive pages, collect traces/HAR/video/screenshot [page:0][web:11] |
| UI | React + TypeScript + Framer Motion | Project/workspace UI, test editor, run dashboard, debugging console |
| Storage | SQLite + filesystem workspace | Persist projects, scripts, configs, runs, artifacts |
| “Script ingestion” | Go | Import from paste/file/git URL, parse userscript metadata, version, provenance |

### 4.2 Process model
- UI talks to Runner over `http://127.0.0.1:<port>` with WebSocket streaming for logs/events (near real-time run console).  
- Runner launches **persistent** browser contexts for extension testing (per Playwright guidance). [page:0]
- Each run produces a run folder with normalized artifact paths and a single `run.json` manifest that the UI can load.

## 5) Core workflows

### 5.1 Script ingestion (options required)
| Input type | UX | Backend behavior |
|---|---|---|
| Paste script | Editor textarea, “Import” | Save to workspace, hash, parse metadata |
| Local file | File picker | Copy into workspace, preserve original name |
| Git URL | “Add from Git” dialog | Clone shallow (or fetch), select file(s), record commit SHA |

**Userscript metadata parsing (required):**
- Parse `// ==UserScript==` block; extract `@name`, `@namespace`, `@version`, `@match`, `@include`, `@exclude`, `@run-at`, and any GM permissions.
- Store the parsed metadata alongside the exact raw script content (for reproducibility).

### 5.2 Engine selection (settings)
Settings must allow:
- Userscript engine: `Tampermonkey` or `Violentmonkey` (single-select). [web:45][web:50]
- Browser: `Chromium (Playwright bundled)` default; other “channels” allowed only if compatible with extension loading constraints. [page:0]
- Headless mode: default ON where supported; fall back to headed if extension mode requires it, with clear UI warning. [page:0]

### 5.3 Installing engine + script into the browser profile
**Requirement:** install the chosen extension into the persistent profile and ensure the chosen script is enabled for the test URL(s). [page:0][web:45]

**Recommended approach (robustness > cleverness):**
- Bundle known-good extension builds for TM/VM into the app (or a controlled “extension cache” directory) so runs are reproducible and do not rely on web store downloads. (This also reduces remote-code ambiguity under MV3 policies.) [web:41]
- For each run, create a new user data dir (profile) unless user explicitly opts into “reuse profile” for faster iteration. [page:0]

**Note (explicitly a risk/guess):**
- Automating import/enable flows inside TM/VM UIs can be brittle because extension UIs change.  
- v1 should support at least one deterministic install mechanism (e.g., opening a local `.user.js` page and confirming “Install” via the extension UI), and treat selectors as version-pinned.  
(We will validate feasibility during the spike milestone.)

## 6) Playwright-Go runner requirements

### 6.1 Playwright-Go versioning & installation
- Runner must pin Playwright-Go and install the matching driver/browsers as recommended by the project docs (minor versions must match). [web:11][web:9]
- Runner must expose a “Doctor” command that verifies driver, browsers, and required dependencies are installed.

### 6.2 Browser launch strategy (extensions)
- Use Chromium with a **persistent context** and pass `--disable-extensions-except` and `--load-extension` as shown by Playwright’s extension docs. [page:0]
- Must be able to retrieve MV3 service worker and derive `extensionId` from the service worker URL, to open internal pages like `chrome-extension://<id>/...` when needed. [page:0]

### 6.3 Observability hooks (must capture)
| Signal | How | Notes |
|---|---|---|
| Console logs | Subscribe to context/page console events; store structured logs | Playwright supports browserContext `on('console')` events. [page:1] |
| Unhandled errors | Capture page errors + context “weberror” | Playwright exposes context “weberror” events. [page:1] |
| Requests/responses | Capture request/response events; optionally redact | Playwright supports context request/response events. [page:1] |
| Trace | Enable tracing around tests; save `trace.zip` | Trace Viewer exists for inspecting recorded traces. [web:51][web:52] |
| HAR | Support recording/replay via HAR in advanced mode | Playwright documents HAR usage and routing from HAR. [page:1][web:60] |
| Storage state | Snapshot cookies/localStorage/IndexedDB when requested | BrowserContext storageState can include IndexedDB. [page:1] |

## 7) Validation & checklist flow (user-facing)

### 7.1 Checklist UI model
Each “Run” has a checklist composed of items that can be toggled on/off per suite.

Checklist categories (all supported):
- DOM assertions
- Visual regression (screenshots)
- Network assertions (requests/responses + HAR)
- Behavioral flow (multi-step UX)
- Performance smoke checks (basic timings; optional v1)

### 7.2 Checklist item schema (example)
Each checklist item has:
- `id`, `title`, `type`, `enabled`
- `setup` (preconditions)
- `steps` (actions)
- `assertions`
- `artifacts` expected (e.g., screenshot name)
- `severity` (blocker/warn/info)

### 7.3 Example checklist (GFM task list)
- [ ] DOM: Ensure header text equals expected
- [ ] DOM: Ensure script-added UI exists and is clickable
- [ ] Visual: Capture baseline screenshot and diff
- [ ] Network: Assert no requests to blocked hosts
- [ ] Network: Record HAR for run (optional) [web:60]
- [ ] Flow: Login → navigate → action → verify end state

## 8) UI/UX requirements (React + TS + Framer)

### 8.1 Layout (pro workflow)
| Screen | Must include |
|---|---|
| Projects | Workspaces, scripts, suites |
| Script detail | Raw editor, parsed metadata, provenance (paste/file/git + SHA), enable/disable |
| Suite editor | URLs, engine setting (TM/VM), checklist composition, timeouts, artifacts config |
| Run dashboard | Timeline, pass/fail per checklist item, artifact gallery, “Open trace” button [web:51][web:52] |
| Debug console | Filterable logs (runner + browser console + network), copy/export, correlated by run |

### 8.2 Debugging console (requirements)
- One unified stream that merges:
  - Runner logs
  - Browser console logs
  - Page errors/weberrors [page:1]
  - Request/response summaries (method, URL, status)
- Correlation IDs:
  - `run_id`, `suite_id`, `script_id`
  - `page_id` / `frame_id` when possible
- “Evidence panel” to open:
  - trace.zip in Trace Viewer [web:51][web:52]
  - screenshots/videos
  - HAR viewer (basic) [web:60]

## 9) Logging, artifacts, and troubleshooting

### 9.1 Artifact directory contract
Each run creates:
- `runs/<run_id>/run.json` (manifest)
- `runs/<run_id>/logs/runner.ndjson`
- `runs/<run_id>/logs/browser-console.ndjson`
- `runs/<run_id>/artifacts/trace.zip` (if enabled) [web:51][web:52]
- `runs/<run_id>/artifacts/*.png` (screenshots)
- `runs/<run_id>/artifacts/network.har` (if enabled) [web:60]
- `runs/<run_id>/artifacts/storageState.json` (if enabled) [page:1]

### 9.2 Troubleshooting principles
- Default to *structured* logs (NDJSON) with stable keys, not free-form strings.
- Make every checklist assertion attach evidence (selector snapshot, screenshot, network row, trace timestamp).

## 10) Security and safety
- Treat all imported userscripts as untrusted code; clearly warn users that scripts can access page data and can exfiltrate information.  
- Never automatically run scripts against user-specified domains without explicit user confirmation per suite.
- Provide redaction rules for logs (tokens, cookies, headers) and a “safe export” mode.

## 11) Public API / CLI (for CI & power users)
### 11.1 CLI commands (Go)
| Command | Purpose |
|---|---|
| `lab doctor` | Validate Playwright-Go + browser binaries installed [web:11] |
| `lab run --suite <id>` | Execute suite; exit code indicates pass/fail |
| `lab export --run <id>` | Bundle artifacts (zip) |
| `lab list` | List suites/scripts |
| `lab serve` | Run the local API for the UI |

### 11.2 Local API
- `GET /v1/projects`
- `POST /v1/scripts/import` (paste/file/git URL)
- `POST /v1/runs` (start run)
- `GET /v1/runs/:id` (status + results)
- `WS /v1/runs/:id/logs` (stream)

## 12) Compatibility matrix (must be shown in UI)
| Engine | Browser | MV3 status | Notes |
|---|---|---|---|
| Tampermonkey | Chromium (Playwright bundled) | Supported; TM has MV3 versions and MV3 limitations | Example: MV3 gaps like `GM_webRequest` not supported yet. [web:45] |
| Violentmonkey | Chromium | Risky/possibly blocked depending on MV2 deprecation | VM states not supported on Chrome due to MV2 architecture. [web:50] |
| Violentmonkey | Firefox | Likely viable path | (Validate during spike; keep version-pinned.) |

## 13) Acceptance criteria (MVP)
| Area | Criteria |
|---|---|
| Engine load | App can launch Chromium persistent context with the chosen extension reliably. [page:0] |
| Script install | User can import a userscript via paste and run it on a target URL with engine enabled. |
| Validation | User can run at least: 1 DOM assertion, 1 screenshot capture, 1 network assertion, 1 multi-step flow in a single suite. |
| Observability | Run produces trace.zip and can be opened in Trace Viewer, plus NDJSON logs in UI. [web:51][web:52] |
| Settings | TM vs VM is selectable; UI clearly warns about VM/Chromium MV2 constraints. [web:50] |

## 14) Rough cost envelope (USD, ballpark)
- Local dev: $0 incremental compute (uses developer machine).
- CI (later): ~$0–$50/month depending on GitHub Actions minutes and artifact retention.
- Optional AI features (if added later): variable; depends on Claude model + tokens; plan to gate behind a toggle and per-run budget.
