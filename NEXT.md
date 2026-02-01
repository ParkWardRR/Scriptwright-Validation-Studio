# What's Next — Roadmap & Remediation Plan

**Project Status:** 45% complete, working prototype
**Security Status:** ⚠️ 3 CRITICAL vulnerabilities (must fix before production)
**Priority:** Security fixes → Reliability → Features

---

## 📊 Current State

### ✅ What Works Today
- Load userscripts (file, URL, Git)
- Run in Chromium (via Playwright)
- Capture screenshots, videos, logs
- Test DOM actions (click, fill, wait, assert)
- Visual regression (pixel diffs)
- Network assertions (blocked hosts, status codes)
- Web UI and CLI
- Server deployment

### ❌ What Doesn't Work
- Reliable Tampermonkey/Violentmonkey loading
- Project/suite persistence
- Real-time log streaming
- Embedded trace/HAR viewers
- Concurrent test execution
- CI/CD integration

### 🚨 Security Issues (CRITICAL)
1. **Path Traversal** in extension upload → Arbitrary file write
2. **Command Injection** in git/ffmpeg → Remote code execution
3. **No Authentication** on REST API → Unauthenticated access

**⚠️ DO NOT use in production until Phase 1 security fixes are applied!**

---

## 🎯 Week-by-Week Plan

### Week 1: CRITICAL Security Fixes (MUST DO FIRST)

**Priority:** Blocking for production
**Estimated Time:** 3-4 days

#### Day 1-2: Fix Security Vulnerabilities

**1.1 Path Traversal Fix**
```go
// File: cmd/lab/main.go:262-267
// BEFORE (VULNERABLE):
name := header.Filename
dest := filepath.Join("extensions", name)
out, err := os.Create(dest)

// AFTER (SECURE):
func sanitizeFilename(filename string) (string, error) {
    base := filepath.Base(filename)
    if strings.Contains(base, "..") || strings.HasPrefix(base, ".") {
        return "", fmt.Errorf("invalid filename: %s", filename)
    }
    ext := filepath.Ext(base)
    allowedExts := map[string]bool{".crx": true, ".xpi": true, ".zip": true}
    if !allowedExts[ext] {
        return "", fmt.Errorf("invalid extension: %s", ext)
    }
    hash := sha256.Sum256([]byte(base + time.Now().String()))
    return hex.EncodeToString(hash[:8]) + ext, nil
}
```

**1.2 Command Injection Fix**
```go
// File: internal/runner/runner.go:823
// Add validation before git clone
func validateGitURL(repo string) error {
    u, err := url.Parse(repo)
    if err != nil {
        return fmt.Errorf("invalid git URL: %w", err)
    }
    if u.Scheme != "https" && u.Scheme != "http" {
        return fmt.Errorf("unsupported scheme: %s", u.Scheme)
    }
    dangerous := []string{"|", ";", "&", "$", "`", "\n"}
    for _, char := range dangerous {
        if strings.Contains(repo, char) {
            return fmt.Errorf("invalid characters in URL")
        }
    }
    return nil
}
```

**1.3 Add Authentication**
```go
// File: cmd/lab/main.go
// Add middleware
func (a *authMiddleware) authenticate(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if a.apiKey == "" {
            next(w, r)
            return
        }
        auth := r.Header.Get("Authorization")
        if !strings.HasPrefix(auth, "Bearer ") {
            writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
            return
        }
        token := strings.TrimPrefix(auth, "Bearer ")
        if subtle.ConstantTimeCompare([]byte(token), []byte(a.apiKey)) != 1 {
            writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid key"})
            return
        }
        next(w, r)
    }
}

// Usage: export API_KEY="your-secret-here"
```

#### Day 3: Testing & Validation
- Test path traversal attempts (should fail)
- Test command injection (should fail)
- Test authenticated/unauthenticated requests
- Security scan

#### Day 4: Deploy Security Fixes
- Commit changes
- Deploy to production
- Update documentation

**Acceptance Criteria:**
- ✅ All file uploads sanitized
- ✅ All external commands validated
- ✅ API requires authentication
- ✅ Security scan passes

---

### Week 2: Quick Wins (Reliability)

**Priority:** High value, low effort
**Estimated Time:** 4-5 days

#### Day 1: Fix Trace Timing Bug
```go
// Move trace Start BEFORE page operations
if opts.CaptureTrace {
    ctx.Tracing().Start(...)  // Before navigate
}
// ... page operations ...
if opts.CaptureTrace {
    ctx.Tracing().Stop(...)   // Before page.Close()
}
```

#### Day 2: Add `lab doctor` Command
```bash
lab doctor
# Checks:
# - Playwright binary exists
# - Chromium installed
# - ffmpeg available
# - webp tools available
```

#### Day 3: Improve Extension Logging
- Add clear logs when extension not found
- Show fallback to init-script
- Better error messages

#### Day 4-5: UI Improvements
- Add HAR/Trace download buttons
- Show run duration
- Add "Copy curl command" button

**Acceptance Criteria:**
- ✅ Trace files captured correctly
- ✅ `lab doctor` validates installation
- ✅ Clear extension status logs
- ✅ UI improvements deployed

---

### Week 3-4: Architecture Refactoring

**Priority:** Medium (improves maintainability)
**Estimated Time:** 8-10 days

#### Extract Components

Break up monolithic `Run()` function (282 lines → multiple files):

```
internal/runner/
├── runner.go          # Orchestrator (100 lines)
├── script_loader.go   # Load scripts
├── engine_manager.go  # TM/VM installation
├── browser_manager.go # Playwright lifecycle
├── flow_executor.go   # Execute steps
├── artifact_collector.go  # Screenshots, videos
├── visual_differ.go   # Visual regression
└── manifest_builder.go    # Build results
```

**New Architecture:**
```go
type Runner struct {
    scriptLoader      *ScriptLoader
    engineManager     *EngineManager
    browserManager    *BrowserManager
    flowExecutor      *FlowExecutor
    artifactCollector *ArtifactCollector
    visualDiffer      *VisualDiffer
    manifestBuilder   *ManifestBuilder
    logger            Logger
}

func (r *Runner) Run(ctx context.Context, opts Options) (Result, error) {
    script, err := r.scriptLoader.Load(ctx, opts)
    if err != nil {
        return Result{}, fmt.Errorf("load script: %w", err)
    }
    browser, err := r.browserManager.Launch(ctx, opts)
    if err != nil {
        return Result{}, fmt.Errorf("launch browser: %w", err)
    }
    defer browser.Close()
    // ... etc
}
```

#### Error Handling

**Replace:**
```go
_ = os.RemoveAll(dir)  // BAD: Silent failure
```

**With:**
```go
if err := os.RemoveAll(dir); err != nil {
    logger.Warn("cleanup failed", "dir", dir, "error", err)
}
```

#### Concurrency Safety

Add job queue for concurrent runs:
```go
type JobQueue struct {
    jobs    chan Job
    results map[string]chan Result
    workers int
}

func (q *JobQueue) Submit(id string, opts Options) <-chan Result {
    resultChan := make(chan Result, 1)
    q.results[id] = resultChan
    q.jobs <- Job{ID: id, Options: opts}
    return resultChan
}
```

**Acceptance Criteria:**
- ✅ Run() function < 150 lines
- ✅ No functions > 100 lines
- ✅ Error handling consistent
- ✅ Concurrent requests safe

---

### Week 5: Observability & Testing

**Priority:** Medium
**Estimated Time:** 5-7 days

#### Structured Logging
```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(logFile, nil))
logger.Info("run started",
    "run_id", runID,
    "url", opts.TargetURL,
    "engine", opts.Engine,
)
```

#### Request Logging
```go
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next(w, r)
        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "duration_ms", time.Since(start).Milliseconds(),
        )
    }
}
```

#### Unit Tests (Target: 60% Coverage)
```go
func TestScriptLoader_LoadFromFile(t *testing.T) {
    loader := NewScriptLoader()
    tmpfile, _ := os.CreateTemp("", "test-*.user.js")
    defer os.Remove(tmpfile.Name())

    script, err := loader.Load(context.Background(), Options{
        ScriptPath: tmpfile.Name(),
    })
    assert.NoError(t, err)
    assert.NotEmpty(t, script.Content)
}
```

**Acceptance Criteria:**
- ✅ Structured logging (slog)
- ✅ Request/response logging
- ✅ Unit test coverage ≥ 60%
- ✅ Integration tests for happy path

---

## 🚀 Medium-Term Features (Weeks 6-10)

### Real-Time Log Streaming (Week 6)
- **Effort:** 3-5 days
- **Tech:** WebSocket
- **Benefit:** Better debugging

### Extension Bundling (Week 7)
- **Effort:** 3-4 days
- **Benefit:** One-click setup, reproducible

### Formal Checklist Schema (Week 8)
- **Effort:** 4-5 days
- **Benefit:** Reusable test suites

### Export Bundles (Week 9)
- **Effort:** 1-2 days
- **Benefit:** Share results, CI integration

### GitHub Actions Template (Week 10)
- **Effort:** 2-3 weeks
- **Benefit:** Automated testing

---

## 📈 Long-Term Features (Months 3-6)

### React + TypeScript UI Rewrite
- **Effort:** 6-8 weeks
- **Recommendation:** Wait until 10+ active users

### SQLite Persistence Layer
- **Effort:** 4-6 weeks
- **Benefit:** Save projects, suites, history
- **Recommendation:** Wait until 5+ active users

### Embedded Trace/HAR Viewers
- **Effort:** 4-5 weeks
- **Priority:** Medium (nice to have)

### Enterprise Features
- Multi-user support
- SSO/RBAC
- Audit logs
- Distributed execution

**Not needed for MVP.** Focus on core functionality first.

---

## 🎯 This Week's Action Items

**For production readiness:**

1. **Fix path traversal vulnerability** (4 hours)
2. **Fix command injection vulnerability** (4 hours)
3. **Add authentication middleware** (4 hours)
4. **Security testing** (4 hours)
5. **Deploy security fixes** (2 hours)

**Total: ~18 hours (2-3 days)**

After this week, the app will be:
- ✅ Secure for production use
- ✅ Protected against major attacks
- ✅ Ready for wider deployment

---

## 📋 Detailed Implementation Guide

### Security Fix Checklist

**Path Traversal:**
- [ ] Add `sanitizeFilename()` function
- [ ] Validate file extensions (.crx, .xpi, .zip only)
- [ ] Use absolute paths
- [ ] Check destination is within allowed directory
- [ ] Add file size limits
- [ ] Test with malicious filenames

**Command Injection:**
- [ ] Add `validateGitURL()` function
- [ ] Whitelist HTTPS/HTTP schemes only
- [ ] Reject dangerous characters
- [ ] Use `CommandContext` with timeout
- [ ] Clear environment variables
- [ ] Validate ffmpeg/webp paths
- [ ] Test with injection attempts

**Authentication:**
- [ ] Create auth middleware
- [ ] Add API key validation (constant-time compare)
- [ ] Protect sensitive endpoints (/v1/runs, /v1/extensions)
- [ ] Leave /health and /ui/ public
- [ ] Add rate limiting (10 req/min)
- [ ] Add request logging
- [ ] Update documentation

---

## 🔍 Code Quality Standards

**For all new code:**
- [ ] No function > 100 lines
- [ ] Cyclomatic complexity < 15
- [ ] All errors checked (no `_` blank assignments)
- [ ] Errors wrapped with context
- [ ] Unit tests for business logic
- [ ] Integration tests for critical paths
- [ ] Structured logging (slog)
- [ ] No sensitive data in logs
- [ ] Public functions documented

---

## 📊 Success Metrics

**Before Remediation:**
- Security: D (3 critical vulnerabilities)
- Maintainability: D (god functions)
- Test Coverage: 5%

**After Phase 1 (Week 1):**
- Security: B (critical issues fixed)
- Maintainability: D (unchanged)
- Test Coverage: 5%

**After Phase 2-3 (Week 5):**
- Security: A (all issues fixed)
- Maintainability: B (refactored)
- Test Coverage: 60%+

**Key Performance Indicators:**
- Time to add feature: 50% reduction
- Mean time to debug: 70% reduction
- Security incidents: 0
- Production uptime: 99%+

---

## 🤔 Decision Guide

### Should You Fix Security Issues First?
**Answer: YES!** (Non-negotiable)
- Current state: Vulnerable to attacks
- Impact: Critical (data loss, unauthorized access)
- Effort: Low (2-3 days)

### Should You Rewrite the UI?
**Answer: Not yet**
- Vanilla UI works for alpha
- React rewrite = 6-8 weeks
- Wait until 10+ active users
- **Recommendation:** Focus on reliability first

### Should You Add Violentmonkey?
**Answer: Low priority**
- VM doesn't work on Chromium (MV2 deprecated)
- Requires Firefox setup
- **Recommendation:** Focus on Tampermonkey first

### Should You Build Persistence Now?
**Answer: Wait**
- 4-6 weeks effort
- Only needed for multi-user
- **Recommendation:** Wait until 5+ active users

---

## 📚 Additional Resources

### Testing
```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# With race detection
go test -race ./...
```

### Security Scanning
```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run security scan
gosec ./...
```

### Deployment
```bash
# Deploy security fixes
./deploy.sh

# Check service status
ssh -i ~/.ssh/scriptwright alfa@scriptwright "sudo systemctl status userscript-lab"
```

---

## 🎯 Summary: What to Do Next

1. **This Week:** Security fixes (CRITICAL)
2. **Next Week:** Quick wins (reliability)
3. **Weeks 3-4:** Architecture refactoring
4. **Week 5:** Testing & observability
5. **Weeks 6-10:** Medium-term features
6. **Months 3-6:** Long-term features (if needed)

**Focus:** Security → Reliability → Features

The app works but needs security fixes before production use. Following this plan gets it production-ready in ~5 weeks.

---

**Document Version:** 2.0 (Combined NEXT.md + REMEDIATION.md)
**Last Updated:** 2026-01-31
