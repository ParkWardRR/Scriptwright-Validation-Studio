# What's Next — Priority Roadmap

**Status:** Project is ~45% complete. Core functionality works, but advanced features are missing.

---

## Current State (What Works Today)

✅ **You can:**
- Load a userscript from file, URL, or Git
- Run it in Chromium (via Playwright)
- Get screenshots, videos, and logs
- Test DOM actions (click, fill, wait, assert)
- Check visual regressions (pixel diffs)
- Inspect network traffic (blocked hosts, status codes)
- Use the web UI or CLI
- Deploy to a server

❌ **You cannot:**
- Automatically load Tampermonkey/Violentmonkey reliably
- Save projects/suites for reuse
- Stream logs in real-time
- View traces/HAR files in the UI
- Run multiple tests concurrently
- Export test bundles for CI
- Retry failed tests automatically

---

## Priority Fixes (Critical — Do First)

### 1. Fix Trace Capture Timing Bug
**Problem:** Trace starts/stops AFTER page closes, so artifacts may not be captured.
**Impact:** High (trace files might be empty)
**Effort:** Low (1-2 hours)
**Action:**
```go
// In runner.go, move trace Start before page operations
if opts.CaptureTrace {
    ctx.Tracing().Start(...) // Move this BEFORE navigate
}
// ... page operations ...
if opts.CaptureTrace {
    ctx.Tracing().Stop(...)  // Keep this BEFORE page.Close()
}
```

### 2. Add `lab doctor` Command
**Problem:** No way to validate Playwright installation or dependencies.
**Impact:** Medium (users confused by "browser not found" errors)
**Effort:** Low (2-3 hours)
**Action:**
```bash
# Should check:
# - Playwright binary exists
# - Chromium browser installed
# - ffmpeg available (for video)
# - webp tools available
lab doctor
```

### 3. Improve Extension Detection Logging
**Problem:** Extension loading fails silently, user doesn't know why.
**Impact:** High (users think it's working when it's not)
**Effort:** Low (1 hour)
**Action:** Add clear logs when extension not found, falls back to init-script.

---

## Quick Wins (High Value, Low Effort)

### 4. Add HAR/Trace Download Buttons in UI
**Problem:** Links exist but not obvious.
**Impact:** Medium
**Effort:** Low (1 hour)
**Action:** Add prominent "Download HAR" / "Download Trace" buttons in artifact panel.

### 5. Show Run Duration in UI
**Problem:** Manifest has duration_sec but UI doesn't display it.
**Impact:** Low (nice to have)
**Effort:** Low (30 min)
**Action:** Add duration to manifest info panel.

### 6. Add "Copy curl command" Button
**Problem:** Users don't know how to reproduce runs via API.
**Impact:** Medium
**Effort:** Low (1 hour)
**Action:** Generate curl command from current settings, let user copy.

---

## Medium-Term Features (1-2 Weeks Each)

### 7. Real-Time Log Streaming
**Problem:** Logs are static files, not live.
**Solution:** WebSocket streaming
**Effort:** Medium (3-5 days)
**Benefits:**
- See logs as they happen
- Better debugging experience
- Feels more responsive

### 8. Extension Bundling System
**Problem:** Extensions must be manually downloaded/unzipped.
**Solution:** Auto-download and cache extensions
**Effort:** Medium (3-4 days)
**Benefits:**
- One-click setup
- Reproducible runs
- Version pinning

### 9. Formal Checklist Schema
**Problem:** Flow steps are ad-hoc, no validation schema.
**Solution:** Define JSON schema for test checklists
**Effort:** Medium (4-5 days)
**Benefits:**
- Reusable test suites
- Validation before run
- Better error messages

---

## Major Features (4-6 Weeks Each)

### 10. React + TypeScript UI Rewrite
**Problem:** Current UI is vanilla HTML/JS, hard to extend.
**Solution:** Rebuild in React + TypeScript
**Effort:** High (6-8 weeks)
**Benefits:**
- Better UX (animations, responsiveness)
- Easier to add features
- Component reuse

**Not a priority right now.** The vanilla UI works for MVP.

### 11. SQLite Persistence Layer
**Problem:** No way to save projects, suites, or configs.
**Solution:** Add SQLite for structured storage
**Effort:** High (4-6 weeks)
**Benefits:**
- Save/load test suites
- Project organization
- Run history

**Medium priority.** Needed for multi-user or team usage.

### 12. Embedded Trace/HAR Viewers
**Problem:** Trace/HAR files download, not viewable in-app.
**Solution:** Embed viewers (iframe or component)
**Effort:** High (4-5 weeks)
**Benefits:**
- No external tools needed
- Better debugging workflow

**Medium priority.** Nice to have, not critical.

---

## CI/CD Integration (Future)

### 13. GitHub Actions Template
**Effort:** Medium (2-3 weeks)
**Requires:** CLI export command, deterministic runs
**Benefits:** Run tests on PRs, artifact upload

### 14. Export Bundles
**Effort:** Low (1-2 days)
**Benefits:** Share test results, archive runs

---

## Immediate Next Steps (This Week)

**For a working, useful tool:**

1. **Fix trace timing** (1-2 hours)
2. **Add `lab doctor`** (2-3 hours)
3. **Improve extension logging** (1 hour)
4. **Test Tampermonkey loading with actual binary** (3-4 hours)

**Total:** ~1 day of work

After that, the tool will be:
- ✅ Stable for basic use
- ✅ Good for demos/prototyping
- ✅ Deployable to teams

---

## What Should You Build First?

**If you want:**
- **Reliability:** Fix trace bug, add `lab doctor`, test extensions
- **User experience:** Real-time logs, better UI feedback
- **Team usage:** Persistence layer, export bundles
- **CI integration:** GitHub Actions template, retry logic
- **Polish:** React UI, embedded viewers, animations

**Recommendation:** Start with reliability fixes (this week), then decide based on user feedback.

---

## Long-Term Vision (6+ Months)

### Full "Pro Workflow" Features
- Multi-user support
- Run history & comparison
- Performance benchmarks
- Automated flake detection
- Slack/Discord notifications
- Cost tracking (Playwright minutes)

### Enterprise Features
- SSO/RBAC
- Audit logs
- Custom runners (Docker-in-Docker)
- Distributed execution
- Private extension hosting

**Not needed for MVP.** Focus on core functionality first.

---

## Decision Points

### Should You Rewrite the UI?

**Pros:**
- Better UX, easier to extend, modern stack

**Cons:**
- 6-8 weeks of work, delays other features

**Recommendation:** **No, not yet.** The vanilla UI is good enough for alpha. Rewrite when you have users asking for specific features.

### Should You Add Violentmonkey Support?

**Pros:**
- Broader engine coverage

**Cons:**
- VM doesn't work on Chromium (MV2 deprecated), requires Firefox setup

**Recommendation:** **Low priority.** Focus on Tampermonkey (works on Chromium) first.

### Should You Build Persistence Now?

**Pros:**
- Enables reusable test suites, better for teams

**Cons:**
- 4-6 weeks of work, schema design complexity

**Recommendation:** **Wait until you have 5+ active users.** Single runs are fine for prototyping.

---

## Summary: What to Do Next

### This Week (Critical)
1. Fix trace timing bug
2. Add `lab doctor` command
3. Test extension loading with real TM binary

### Next 2 Weeks (High Value)
4. Real-time log streaming (WebSocket)
5. Extension bundling system
6. Better error messages

### Next Month (Polish)
7. Formal checklist schema
8. Export bundles
9. GitHub Actions template

### Someday (Not Urgent)
- React UI rewrite
- Persistence layer
- Embedded viewers
- CI/CD integration

**Focus:** Get the core loop (load → test → view results) rock-solid before adding advanced features.
