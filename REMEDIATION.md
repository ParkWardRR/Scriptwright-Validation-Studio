# Architecture Remediation Plan

**Project:** Scriptwright Validation Studio (Philadelphia)
**Current State:** 45% complete, working prototype with significant architectural debt
**Goal:** Production-ready, secure, maintainable application

---

## Executive Summary

### Critical Findings

The application has **3 CRITICAL security vulnerabilities** and **7 HIGH-priority architectural issues** that must be addressed before production use:

🚨 **CRITICAL Security Issues:**
1. Path traversal vulnerability in extension upload (arbitrary file write)
2. Command injection vulnerability in git/ffmpeg execution
3. No authentication on REST API (unauthenticated code execution)

⚠️ **HIGH Priority Issues:**
4. Monolithic 282-line `Run()` function (unmaintainable)
5. Inconsistent error handling (silent failures)
6. No concurrency safety (race conditions)
7. Missing resource limits (DoS vulnerability)

📊 **Overall Code Health:**
- **Lines of Code:** 1,252 (Go)
- **Test Coverage:** ~5% (1 test file only)
- **Cyclomatic Complexity:** Very High (114 if-statements in runner.go)
- **Security Rating:** D (multiple critical vulnerabilities)
- **Maintainability:** D (god functions, tight coupling)

---

## Remediation Roadmap

### Phase 1: Critical Security Fixes (Week 1)

**Estimated Time:** 3-4 days
**Priority:** CRITICAL
**Must complete before any production deployment**

#### 1.1 Fix Path Traversal Vulnerability

**File:** `cmd/lab/main.go:262-267`

**Current Code (VULNERABLE):**
```go
name := header.Filename
dest := filepath.Join("extensions", name)
out, err := os.Create(dest)
```

**Fixed Code:**
```go
import (
    "path/filepath"
    "strings"
    "crypto/sha256"
    "encoding/hex"
)

func sanitizeFilename(filename string) (string, error) {
    // Strip directory components
    base := filepath.Base(filename)

    // Reject suspicious patterns
    if strings.Contains(base, "..") || strings.HasPrefix(base, ".") {
        return "", fmt.Errorf("invalid filename: %s", filename)
    }

    // Validate extension
    ext := filepath.Ext(base)
    allowedExts := map[string]bool{".crx": true, ".xpi": true, ".zip": true}
    if !allowedExts[ext] {
        return "", fmt.Errorf("invalid extension: %s", ext)
    }

    // Generate safe name: hash + original extension
    hash := sha256.Sum256([]byte(base + time.Now().String()))
    safeName := hex.EncodeToString(hash[:8]) + ext

    return safeName, nil
}

// In handleExtensions:
safeName, err := sanitizeFilename(header.Filename)
if err != nil {
    writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
    return
}

// Use absolute path
extDir := filepath.Join(s.workspace, "extensions")
if err := os.MkdirAll(extDir, 0o755); err != nil {
    writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
    return
}

dest := filepath.Join(extDir, safeName)

// Validate destination is within extensions directory
absExt, _ := filepath.Abs(extDir)
absDest, _ := filepath.Abs(dest)
if !strings.HasPrefix(absDest, absExt) {
    writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
    return
}
```

**Additional Hardening:**
- Add file size limit (currently 32MB, validate it)
- Scan uploaded files for malware signatures
- Store metadata (original filename, upload time, SHA256)

**Testing:**
```bash
# Test cases
curl -X POST http://localhost:8787/v1/extensions -F "file=@../../../../etc/passwd"  # Should fail
curl -X POST http://localhost:8787/v1/extensions -F "file=@malicious.exe"          # Should fail
curl -X POST http://localhost:8787/v1/extensions -F "file=@valid.crx"              # Should succeed
```

---

#### 1.2 Fix Command Injection Vulnerability

**File:** `internal/runner/runner.go:429, 446, 823`

**Current Code (VULNERABLE):**
```go
cmd := exec.Command("git", "clone", "--depth", "1", repo, dir)
// repo is user input!
```

**Fixed Code:**
```go
import (
    "net/url"
    "regexp"
)

func validateGitURL(repo string) error {
    // Parse as URL
    u, err := url.Parse(repo)
    if err != nil {
        return fmt.Errorf("invalid git URL: %w", err)
    }

    // Whitelist schemes
    if u.Scheme != "https" && u.Scheme != "http" {
        return fmt.Errorf("unsupported git scheme: %s (use https)", u.Scheme)
    }

    // Reject local paths and file:// URLs
    if u.Scheme == "file" || strings.HasPrefix(repo, "/") || strings.HasPrefix(repo, ".") {
        return fmt.Errorf("local git repositories not allowed")
    }

    // Reject suspicious patterns
    dangerous := []string{"|", ";", "&", "$", "`", "\n", "\r"}
    for _, char := range dangerous {
        if strings.Contains(repo, char) {
            return fmt.Errorf("invalid characters in git URL")
        }
    }

    return nil
}

// In Run():
if opts.ScriptGitRepo != "" {
    if err := validateGitURL(opts.ScriptGitRepo); err != nil {
        return Result{}, fmt.Errorf("git validation failed: %w", err)
    }

    // Use context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", opts.ScriptGitRepo, tempDir)
    cmd.Env = []string{} // Clear environment

    var stderr bytes.Buffer
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return Result{}, fmt.Errorf("git clone failed: %w\nstderr: %s", err, stderr.String())
    }
}
```

**For ffmpeg/img2webp:**
```go
func sanitizeVideoPath(path string) error {
    // Must be within workspace
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }

    absWorkspace, _ := filepath.Abs(".")
    if !strings.HasPrefix(absPath, absWorkspace) {
        return fmt.Errorf("video path outside workspace")
    }

    // Must be .webm file
    if filepath.Ext(path) != ".webm" {
        return fmt.Errorf("invalid video format: %s", filepath.Ext(path))
    }

    return nil
}

// In convertToWebP():
if err := sanitizeVideoPath(input); err != nil {
    return "", err
}

ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, "ffmpeg", "-i", input, "-c:v", "libwebp", "-y", output)
```

---

#### 1.3 Add Authentication Middleware

**File:** `cmd/lab/main.go`

**New Middleware:**
```go
import (
    "crypto/subtle"
    "encoding/base64"
)

type authMiddleware struct {
    apiKey string
}

func newAuthMiddleware() *authMiddleware {
    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        log.Println("WARNING: No API_KEY set, authentication disabled")
    }
    return &authMiddleware{apiKey: apiKey}
}

func (a *authMiddleware) authenticate(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Skip auth if no key configured (dev mode)
        if a.apiKey == "" {
            next(w, r)
            return
        }

        // Check Authorization header
        auth := r.Header.Get("Authorization")
        if auth == "" {
            writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing Authorization header"})
            return
        }

        // Expect "Bearer <key>"
        const prefix = "Bearer "
        if !strings.HasPrefix(auth, prefix) {
            writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid Authorization format"})
            return
        }

        token := strings.TrimPrefix(auth, prefix)

        // Constant-time comparison
        if subtle.ConstantTimeCompare([]byte(token), []byte(a.apiKey)) != 1 {
            writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
            return
        }

        next(w, r)
    }
}

// In routes():
auth := newAuthMiddleware()
mux.HandleFunc("/v1/runs", auth.authenticate(s.handleRuns))
mux.HandleFunc("/v1/extensions", auth.authenticate(s.handleExtensions))
// Leave /health and /ui/ unauthenticated
```

**Usage:**
```bash
# Set API key
export API_KEY="your-secret-key-here"

# Make authenticated request
curl -H "Authorization: Bearer your-secret-key-here" \
  -X POST http://localhost:8787/v1/runs \
  -d '{"url":"https://example.com", ...}'
```

**Additional Hardening:**
- Add rate limiting (e.g., 10 requests/minute per IP)
- Add request logging with IP, timestamp, endpoint
- Consider JWT tokens for multi-user scenarios
- Add CORS whitelist instead of `*`

---

### Phase 2: Architectural Refactoring (Week 2-3)

**Estimated Time:** 8-10 days
**Priority:** HIGH
**Goal:** Break monolithic code into testable, maintainable components

#### 2.1 Extract Runner Components

**Problem:** `Run()` is 282 lines doing everything.

**Solution:** Create separate components:

```
internal/
├── runner/
│   ├── runner.go          # Orchestrator only (100 lines)
│   ├── script_loader.go   # Load scripts (file, URL, git)
│   ├── engine_manager.go  # Install/configure TM/VM
│   ├── browser_manager.go # Playwright lifecycle
│   ├── flow_executor.go   # Execute steps
│   ├── artifact_collector.go  # Generate screenshots, videos
│   ├── visual_differ.go   # Visual regression
│   └── manifest_builder.go    # Build result manifest
```

**New Architecture:**
```go
// runner.go
type Runner struct {
    scriptLoader    *ScriptLoader
    engineManager   *EngineManager
    browserManager  *BrowserManager
    flowExecutor    *FlowExecutor
    artifactCollector *ArtifactCollector
    visualDiffer    *VisualDiffer
    manifestBuilder *ManifestBuilder
    logger          Logger
}

func (r *Runner) Run(ctx context.Context, opts Options) (Result, error) {
    // 1. Load script
    script, err := r.scriptLoader.Load(ctx, opts)
    if err != nil {
        return Result{}, fmt.Errorf("load script: %w", err)
    }

    // 2. Setup browser
    browser, err := r.browserManager.Launch(ctx, opts)
    if err != nil {
        return Result{}, fmt.Errorf("launch browser: %w", err)
    }
    defer browser.Close()

    // 3. Install engine
    if err := r.engineManager.Install(ctx, browser, opts); err != nil {
        return Result{}, fmt.Errorf("install engine: %w", err)
    }

    // 4. Execute flow
    results, err := r.flowExecutor.Execute(ctx, browser, script, opts)
    if err != nil {
        return Result{}, fmt.Errorf("execute flow: %w", err)
    }

    // 5. Collect artifacts
    artifacts, err := r.artifactCollector.Collect(ctx, browser, results)
    if err != nil {
        return Result{}, fmt.Errorf("collect artifacts: %w", err)
    }

    // 6. Visual diff
    diff, err := r.visualDiffer.Compare(ctx, artifacts, opts)
    if err != nil {
        r.logger.Warn("visual diff failed", "error", err)
    }

    // 7. Build manifest
    manifest := r.manifestBuilder.Build(script, results, artifacts, diff)

    return Result{RunID: results.ID, Manifest: manifest}, nil
}
```

**Benefits:**
- Each component testable independently
- Clear responsibilities
- Easier to mock for tests
- Can parallelize some operations
- Better error isolation

---

#### 2.2 Implement Proper Error Handling

**Pattern to Follow:**
```go
// BAD (current)
_ = os.RemoveAll(dir)  // Silent failure

// GOOD (new)
if err := os.RemoveAll(dir); err != nil {
    logger.Warn("failed to remove temp dir", "dir", dir, "error", err)
    // Continue or return based on severity
}
```

**Error Wrapping:**
```go
import "fmt"

// Wrap errors with context
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

**Error Types:**
```go
// Define custom error types
type ErrValidation struct {
    Field   string
    Message string
}

func (e *ErrValidation) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// Usage
if opts.URL == "" {
    return Result{}, &ErrValidation{Field: "URL", Message: "required"}
}
```

**Error Recovery:**
```go
// Add defer-recover for critical sections
func (r *Runner) Run(ctx context.Context, opts Options) (result Result, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic recovered: %v\nstack: %s", r, debug.Stack())
        }
    }()

    // ... rest of function
}
```

---

#### 2.3 Add Concurrency Safety

**Problem:** No locking on shared resources.

**Solution 1: Request Serialization**
```go
type Server struct {
    workspace string
    runMutex  sync.Mutex  // Serialize runs
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
    s.runMutex.Lock()
    defer s.runMutex.Unlock()

    // ... existing code
}
```

**Solution 2: Job Queue (Better)**
```go
import "sync"

type JobQueue struct {
    jobs     chan Job
    results  map[string]chan Result
    mu       sync.RWMutex
    workers  int
}

type Job struct {
    ID      string
    Options Options
}

func NewJobQueue(workers int) *JobQueue {
    q := &JobQueue{
        jobs:    make(chan Job, 100),
        results: make(map[string]chan Result),
        workers: workers,
    }

    for i := 0; i < workers; i++ {
        go q.worker()
    }

    return q
}

func (q *JobQueue) worker() {
    for job := range q.jobs {
        result, err := runner.Run(context.Background(), job.Options)

        q.mu.RLock()
        ch := q.results[job.ID]
        q.mu.RUnlock()

        if ch != nil {
            ch <- result
        }
    }
}

func (q *JobQueue) Submit(id string, opts Options) <-chan Result {
    resultChan := make(chan Result, 1)

    q.mu.Lock()
    q.results[id] = resultChan
    q.mu.Unlock()

    q.jobs <- Job{ID: id, Options: opts}

    return resultChan
}

// In handleRuns:
jobID := uuid.New().String()
resultChan := jobQueue.Submit(jobID, opts)

select {
case result := <-resultChan:
    writeJSON(w, http.StatusOK, result.Manifest)
case <-time.After(5 * time.Minute):
    writeJSON(w, http.StatusRequestTimeout, map[string]string{"error": "run timeout"})
}
```

**Solution 3: Context Cancellation**
```go
// In handleRuns:
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
defer cancel()

result, err := runner.Run(ctx, opts)
```

---

#### 2.4 Add Resource Limits

**Memory Limits:**
```go
// In Containerfile or systemd service
MemoryLimit=2g
MemoryMax=4g
```

**Timeout Configuration:**
```go
type Config struct {
    MaxRunDuration     time.Duration  // Default: 5 minutes
    MaxVideoSize       int64          // Default: 100MB
    MaxScreenshotSize  int64          // Default: 10MB
    MaxHARSize         int64          // Default: 50MB
    MaxConcurrentRuns  int            // Default: 2
}

func (r *Runner) Run(ctx context.Context, opts Options) (Result, error) {
    // Enforce timeout
    ctx, cancel := context.WithTimeout(ctx, r.config.MaxRunDuration)
    defer cancel()

    // Check video size
    if info, err := os.Stat(videoPath); err == nil {
        if info.Size() > r.config.MaxVideoSize {
            return Result{}, fmt.Errorf("video exceeds max size: %d > %d", info.Size(), r.config.MaxVideoSize)
        }
    }
}
```

**Rate Limiting:**
```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiter *rate.Limiter
}

func NewRateLimiter(requestsPerMinute int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(requestsPerMinute)/60, requestsPerMinute),
    }
}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !rl.limiter.Allow() {
            writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
            return
        }
        next(w, r)
    }
}
```

---

### Phase 3: Observability & Logging (Week 4)

**Estimated Time:** 3-4 days
**Priority:** MEDIUM

#### 3.1 Structured Logging

**Replace custom logger with standard library:**
```go
import "log/slog"

// In runner.go:
logger := slog.New(slog.NewJSONHandler(logFile, nil))

logger.Info("run started",
    "run_id", runID,
    "url", opts.TargetURL,
    "engine", opts.Engine,
)

logger.Error("screenshot failed",
    "run_id", runID,
    "error", err,
    "path", screenshotPath,
)
```

**Request Logging Middleware:**
```go
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap ResponseWriter to capture status code
        rw := &responseWriter{ResponseWriter: w, statusCode: 200}

        next(rw, r)

        slog.Info("http request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", rw.statusCode,
            "duration_ms", time.Since(start).Milliseconds(),
            "remote_addr", r.RemoteAddr,
        )
    }
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

---

### Phase 4: Testing (Week 5)

**Estimated Time:** 5-7 days
**Priority:** MEDIUM

#### 4.1 Unit Tests

**Target Coverage:** 60%

**Example: Script Loader Tests**
```go
// internal/runner/script_loader_test.go
func TestScriptLoader_LoadFromFile(t *testing.T) {
    loader := NewScriptLoader()

    // Create temp file
    tmpfile, err := os.CreateTemp("", "test-*.user.js")
    require.NoError(t, err)
    defer os.Remove(tmpfile.Name())

    content := "// ==UserScript==\n// @name Test\n// ==/UserScript==\nconsole.log('test');"
    _, err = tmpfile.WriteString(content)
    require.NoError(t, err)
    tmpfile.Close()

    // Test loading
    script, err := loader.Load(context.Background(), Options{ScriptPath: tmpfile.Name()})
    assert.NoError(t, err)
    assert.Equal(t, "Test", script.Meta.Name)
    assert.Contains(t, script.Content, "console.log")
}

func TestScriptLoader_LoadFromURL(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("// ==UserScript==\n// @name URLTest\n// ==/UserScript=="))
    }))
    defer server.Close()

    loader := NewScriptLoader()
    script, err := loader.Load(context.Background(), Options{ScriptURL: server.URL})
    assert.NoError(t, err)
    assert.Equal(t, "URLTest", script.Meta.Name)
}
```

**Example: Validation Tests**
```go
// cmd/lab/validation_test.go
func TestSanitizeFilename(t *testing.T) {
    tests := []struct {
        input   string
        wantErr bool
    }{
        {"valid.crx", false},
        {"../../../etc/passwd", true},
        {".hidden.crx", true},
        {"test.exe", true},
        {"normal-file.xpi", false},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            _, err := sanitizeFilename(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### 4.2 Integration Tests

**Example: End-to-End Run**
```go
// internal/runner/integration_test.go
func TestRun_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    runner := NewRunner(Config{})

    opts := Options{
        TargetURL: "https://example.com",
        ScriptPath: "../../scripts/wikipedia-dark.user.js",
        Headless: true,
        Engine: "Tampermonkey (init-script)",
        Workspace: t.TempDir(),
    }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    result, err := runner.Run(ctx, opts)
    require.NoError(t, err)

    assert.NotEmpty(t, result.RunID)
    assert.NotEmpty(t, result.Manifest.Screenshot)
    assert.FileExists(t, result.Manifest.Screenshot)
}
```

---

## Implementation Timeline

### Week 1: Critical Security (MUST DO FIRST)
- [ ] Day 1-2: Fix path traversal + add sanitization
- [ ] Day 2-3: Fix command injection + add validation
- [ ] Day 3-4: Add authentication middleware
- [ ] Day 4: Security testing & validation
- [ ] Day 5: Deploy security fixes to production

### Week 2-3: Architecture Refactoring
- [ ] Day 1-3: Extract runner components (script_loader, engine_manager, etc.)
- [ ] Day 4-5: Implement proper error handling
- [ ] Day 6-7: Add concurrency safety (job queue)
- [ ] Day 8-9: Add resource limits & timeouts
- [ ] Day 10: Integration testing

### Week 4: Observability
- [ ] Day 1-2: Replace custom logger with slog
- [ ] Day 2-3: Add request logging middleware
- [ ] Day 3-4: Add metrics collection
- [ ] Day 4: Testing & validation

### Week 5: Testing
- [ ] Day 1-3: Write unit tests (target 60% coverage)
- [ ] Day 4-5: Write integration tests
- [ ] Day 5-7: CI/CD integration

---

## Acceptance Criteria

### Phase 1 (Security) - COMPLETE WHEN:
- ✅ All file uploads sanitized and validated
- ✅ All external commands validated and sandboxed
- ✅ API requires authentication (API key or JWT)
- ✅ CORS restricted to specific origins
- ✅ Rate limiting implemented
- ✅ Security audit passes (automated scan + manual review)
- ✅ No CRITICAL or HIGH vulnerabilities in codebase

### Phase 2 (Architecture) - COMPLETE WHEN:
- ✅ Runner.Run() < 150 lines
- ✅ No functions > 100 lines
- ✅ Cyclomatic complexity < 15 per function
- ✅ All components have clear interfaces
- ✅ Error handling consistent (no blank `_` ignores)
- ✅ Concurrent requests handled safely
- ✅ Resource limits enforced

### Phase 3 (Observability) - COMPLETE WHEN:
- ✅ All logs structured (slog/JSON)
- ✅ Request/response logging on all endpoints
- ✅ Timing metrics captured
- ✅ Error correlation IDs present
- ✅ Log aggregation ready (e.g., can export to Loki/Elasticsearch)

### Phase 4 (Testing) - COMPLETE WHEN:
- ✅ Unit test coverage ≥ 60%
- ✅ All critical paths have tests
- ✅ Integration tests for happy path + common failures
- ✅ CI runs tests automatically
- ✅ Test failures block deployment

---

## Risk Assessment

### High Risk Areas
1. **Backward Compatibility:** Refactoring may break existing API clients
   - **Mitigation:** Version API (v1 → v2), deprecate old endpoints
2. **Performance Regression:** New middleware may slow requests
   - **Mitigation:** Benchmark before/after, optimize hot paths
3. **Migration Complexity:** Refactoring runner is large effort
   - **Mitigation:** Incremental refactoring, feature flags

### Low Risk Areas
- Adding auth (can be optional via env var)
- Fixing security issues (no API changes)
- Adding tests (no runtime impact)

---

## Success Metrics

**Before Remediation:**
- Security: D (3 critical vulnerabilities)
- Maintainability: D (god functions, no tests)
- Test Coverage: 5%
- Error Handling: Poor (silent failures)

**After Remediation:**
- Security: A (no critical/high vulnerabilities)
- Maintainability: B (clear components, interfaces)
- Test Coverage: 60%+
- Error Handling: Good (consistent patterns)

**Key Performance Indicators:**
- Time to add new feature: **50% reduction** (better architecture)
- Mean time to debug: **70% reduction** (better logs, tests)
- Security incidents: **0** (vs. potential now)
- Production uptime: **99%+** (vs. unknown now)

---

## Next Steps

1. **Review this plan** with team/stakeholders
2. **Prioritize** which phases to tackle (recommend: Phase 1 → Phase 2 → Phase 3 → Phase 4)
3. **Create tickets** for each task
4. **Assign ownership** (who will implement each phase?)
5. **Set deadlines** (when does this need to be production-ready?)
6. **Begin Phase 1** (security fixes are critical!)

---

## Appendix: Code Quality Checklist

Use this checklist for code reviews:

**Security:**
- [ ] No user input used directly in commands
- [ ] File paths validated and sanitized
- [ ] Authentication required for sensitive endpoints
- [ ] CORS configured (not *)
- [ ] Rate limiting enabled

**Error Handling:**
- [ ] All errors checked (no blank `_`)
- [ ] Errors wrapped with context
- [ ] Panics recovered in goroutines
- [ ] Defers used for cleanup

**Concurrency:**
- [ ] Shared state protected by mutex
- [ ] Context used for cancellation
- [ ] Goroutines cleaned up properly
- [ ] No race conditions (verified with -race)

**Testing:**
- [ ] Unit tests for business logic
- [ ] Integration tests for happy path
- [ ] Error cases tested
- [ ] Table-driven tests where appropriate

**Logging:**
- [ ] Structured logging (slog)
- [ ] No sensitive data in logs
- [ ] Correlation IDs present
- [ ] Appropriate log levels

**Documentation:**
- [ ] Public functions documented
- [ ] Complex logic explained
- [ ] README updated
- [ ] API changes noted

---

**Document Version:** 1.0
**Last Updated:** 2026-01-31
**Author:** Architecture Review Team
