# Roadmap — Feature Status

**Last Updated:** 2026-01-31
**Overall Progress:** 45% complete

---

## Milestone Overview

| Milestone | Goal | Status | Completion |
|-----------|------|--------|------------|
| M0 — Spike | Prove Playwright + extensions work | 🟡 Partial | 70% |
| M1 — MVP Runner | Core test execution + artifacts | ✅ Done | 90% |
| M2 — UI v1 | Basic web interface | 🟡 Partial | 50% |
| M3 — Testing to Max | Full validation features | 🟡 Partial | 30% |
| M4 — Hardening | Reliability + debugging | ❌ Not Started | 5% |
| M5 — CI & Sharing | Team-scale usage | ❌ Not Started | 0% |

---

## M0 — Spike (Feasibility Test)

**Goal:** Prove we can run userscripts with real browsers and extensions.

### Done ✅
- [x] Playwright-Go integration
- [x] Persistent browser context launch
- [x] Screenshot capture (PNG)
- [x] Video recording (WebM → WebP)
- [x] NDJSON logging
- [x] Userscript metadata parsing
- [x] Sample artifacts (Wikipedia Dark Mode demo)
- [x] Init-script injection works reliably

### Partial 🟡
- [~] Extension loading (attempted but fragile)
  - Playwright args configured (`--load-extension`, `--disable-extensions-except`)
  - Extension ID detection works
  - TM install automation attempted (brittle selectors)
  - VM install not implemented
  - **Blocker:** Requires manual extension download/setup

### Not Done ❌
- [ ] Deterministic TM/VM install flow
- [ ] Bundled extension binaries (not checked in due to size/licensing)
- [ ] 3x local CI validation

**Status:** 70% complete. Core works, extension automation needs improvement.

---

## M1 — MVP Runner (Core Execution)

**Goal:** Reliable test execution with structured output.

### Done ✅
- [x] `lab run` CLI command
- [x] `lab serve` API server
- [x] `lab list` (show previous runs)
- [x] Script ingestion:
  - [x] File path
  - [x] HTTP URL
  - [x] Git repo (clone + extract)
- [x] Artifact generation:
  - [x] Screenshot (PNG)
  - [x] Video (WebP)
  - [x] Logs (NDJSON)
  - [x] Manifest (JSON)
- [x] Profile management (temp directory per run)
- [x] DOM assertions (selector existence checks)

### Not Done ❌
- [ ] `lab doctor` (validate installation)
- [ ] `lab export` (bundle artifacts)
- [ ] Script paste/stdin input (only file/URL/git)

**Status:** 90% complete. Core runner works great.

---

## M2 — UI v1 (Web Interface)

**Goal:** Usable web interface for running tests and viewing results.

### Done ✅
- [x] Web UI prototype (HTML/CSS/JS)
- [x] Settings panel:
  - [x] Target URL
  - [x] Script (file/URL/git)
  - [x] Engine selection
  - [x] Headless toggle
  - [x] HAR/Trace toggles
  - [x] Baseline directory
  - [x] Blocked hosts
  - [x] Visual threshold slider
- [x] Browser pane (embedded iframe)
- [x] Console (log stream from NDJSON file)
- [x] Artifact preview:
  - [x] Screenshot display
  - [x] WebP video display
  - [x] HAR/trace download links
- [x] Flow editor stub (add steps, export JSON)
- [x] API integration (`/v1/runs` endpoint)
- [x] Fallback simulation when API offline

### Partial 🟡
- [~] Real-time log streaming (currently: static file reads)
- [~] Flow step builder (basic HTML, no visual feedback)

### Not Done ❌
- [ ] React + TypeScript (current: vanilla JS)
- [ ] Framer Motion animations
- [ ] Project/suite persistence
- [ ] Timeline visualization
- [ ] Embedded trace/HAR viewers
- [ ] Run history browser
- [ ] Comparison view (diff two runs)

**Status:** 50% complete. Works but not polished.

---

## M3 — Testing to Max (Full Validation)

**Goal:** All validation types (DOM, visual, network, flow) in one run.

### Done ✅
- [x] **DOM Assertions:**
  - [x] Click
  - [x] Fill (form input)
  - [x] Wait (timeout)
  - [x] WaitForSelector
  - [x] Assert text (exact match)
  - [x] Assert contains (substring)
  - [x] Assert exists (selector)
  - [x] Assert not exists
  - [x] Assert attribute value
- [x] **Visual Regression:**
  - [x] Screenshot capture
  - [x] Hash-based baseline comparison
  - [x] Pixel-level diff image (red overlay)
  - [x] Threshold slider (UI)
  - [x] Diff metrics (pixels changed, ratio)
- [x] **Network Assertions:**
  - [x] Blocked hosts detection
  - [x] Status code filtering (4xx/5xx)
  - [x] HAR recording toggle
  - [x] HAR replay toggle (basic)
- [x] **Flow Testing:**
  - [x] Step execution (click, fill, wait, assert)
  - [x] JSON step format
  - [x] UI step editor (basic)

### Partial 🟡
- [~] HAR replay (RouteFromHAR called but minimal error handling)
- [~] Trace capture (timing bug: starts/stops after page.Close)
- [~] Extension loading (TM attempted, VM not implemented)

### Not Done ❌
- [ ] Visual diff preview in UI (currently: image link only)
- [ ] Threshold tuning UI (slider exists but no preview)
- [ ] HAR viewer in UI (currently: download link only)
- [ ] Flow step builder with visual feedback
- [ ] Assertion failure screenshots (capture on error)
- [ ] Performance metrics (timing, memory)
- [ ] 3x local CI validation for TM/VM install

**Status:** 30% complete. Basic features work, advanced features missing.

---

## M4 — Hardening (Reliability)

**Goal:** Reduce flakiness, improve debugging.

### Partial 🟡
- [~] Trace files (captured but timing bug)

### Not Done ❌
- [ ] Retry policy (on failure)
- [ ] Selector diagnostics (when element not found)
- [ ] Redaction rules (tokens, cookies, headers)
- [ ] Export bundles (`lab export` command)
- [ ] Error taxonomy (categorize failures)
- [ ] Flake detection (mark flaky tests)
- [ ] Screenshot on assertion failure
- [ ] Better error messages

**Status:** 5% complete. Minimal error handling.

---

## M5 — CI & Sharing (Team Scale)

**Goal:** Run tests in CI, share results.

### Not Done ❌
- [ ] Headless/headed strategy (flag exists but not fully wired)
- [ ] GitHub Actions template
- [ ] Artifact upload (to GitHub, S3, etc.)
- [ ] JSON report format (manifest exists but not formal report)
- [ ] CLI exit codes (pass/fail)
- [ ] Concurrent run support (job queue)
- [ ] Multi-user support
- [ ] Authentication/authorization

**Status:** 0% complete. Not started.

---

## Feature Breakdown

### Script Ingestion
- ✅ File path
- ✅ HTTP URL
- ✅ Git repo clone
- ❌ Paste/stdin
- ❌ GreasyFork API integration

### Engine Support
- ✅ Init-script injection (works reliably)
- 🟡 Tampermonkey (extension loading attempted, requires manual setup)
- ❌ Violentmonkey (not implemented, MV2 issues on Chromium)

### Artifact Types
- ✅ Screenshot (PNG)
- ✅ Video (WebP)
- ✅ Logs (NDJSON)
- ✅ Manifest (JSON)
- 🟡 HAR (recorded but viewer missing)
- 🟡 Trace (captured but timing bug)
- ❌ Storage state (not implemented)

### Assertions
- ✅ DOM (text, exists, attr)
- ✅ Visual (hash + pixel diff)
- ✅ Network (blocked hosts, status)
- ❌ Performance (timing, memory)
- ❌ Accessibility (WCAG, contrast)

### UI Components
- ✅ Settings panel
- ✅ Browser pane
- ✅ Console
- ✅ Artifact preview
- ✅ Flow editor (basic)
- ❌ Timeline view
- ❌ Comparison view
- ❌ Embedded viewers

### CLI Commands
- ✅ `lab run`
- ✅ `lab serve`
- ✅ `lab list`
- ❌ `lab doctor`
- ❌ `lab export`

### Deployment
- ✅ Container (Containerfile, distroless base)
- ✅ Systemd service
- ✅ Deployment script (`deploy.sh`)
- ✅ Firewall configuration
- ✅ SSH tunnel setup
- ❌ Kubernetes manifests
- ❌ Docker Compose

---

## What's Next?

See [NEXT.md](./NEXT.md) for prioritized next steps.

**Immediate priorities:**
1. Fix trace timing bug
2. Add `lab doctor` command
3. Test extension loading with real TM binary
4. Real-time log streaming

**Medium-term:**
5. Extension bundling system
6. Formal checklist schema
7. Export bundles

**Long-term:**
8. React UI rewrite
9. Persistence layer (SQLite)
10. CI/CD integration

---

## Known Issues

| Issue | Impact | Priority |
|-------|--------|----------|
| Trace timing bug (starts/stops after page.Close) | High | Critical |
| Extension loading fragile (hardcoded selectors) | High | High |
| No real-time log streaming | Medium | Medium |
| HAR replay minimal error handling | Medium | Low |
| No retry logic | Medium | Medium |
| No export bundles | Low | Low |

---

## Decision Log

### Why Vanilla JS Instead of React?
- **Decision:** Build prototype in vanilla HTML/JS first
- **Reason:** Faster to ship MVP, validate product-market fit
- **Trade-off:** Harder to extend, less polished UX
- **Revisit:** When we have 10+ active users

### Why Init-Script Instead of Full Extension Automation?
- **Decision:** Fallback to init-script when extension not found
- **Reason:** Extension UI automation is brittle (selectors change)
- **Trade-off:** Not testing "real" engine behavior
- **Revisit:** When we can bundle extensions reliably

### Why No SQLite Yet?
- **Decision:** Use filesystem only (runs directory)
- **Reason:** Simpler, fewer dependencies, good enough for single-user
- **Trade-off:** No project/suite persistence, harder to query
- **Revisit:** When we need multi-user or team features

---

## Conclusion

**The project is 45% complete.** Core functionality works (run scripts, get artifacts), but advanced features (React UI, persistence, CI) are missing.

**Focus:** Get reliability fixes done (trace bug, extension logging, `lab doctor`) before adding new features.

See [NEXT.md](./NEXT.md) for detailed next steps.
