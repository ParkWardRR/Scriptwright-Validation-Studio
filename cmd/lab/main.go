package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"philadelphia/internal/runner"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "serve":
		serveCmd(os.Args[2:])
	case "list":
		listCmd()
	default:
		usage()
	}
}

func usage() {
	fmt.Println("lab usage:")
	fmt.Println("  lab run   --url <url> --script <path> [--engine <name>] [--ext <dir>] [--headless=false]")
	fmt.Println("  lab serve [--port 8787]")
	fmt.Println("  lab list  # list run ids")
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	url := fs.String("url", "", "Target URL")
	script := fs.String("script", "", "Userscript path")
	engine := fs.String("engine", "Tampermonkey (init-script)", "Engine label")
	ext := fs.String("ext", runner.DiscoverExtensionDir(), "Extension directory (MV3)")
	headless := fs.Bool("headless", true, "Headless mode")
	trace := fs.Bool("trace", false, "Capture trace (stub)")
	har := fs.Bool("har", false, "Capture HAR (stub)")
	replayHar := fs.String("replay-har", "", "Replay from HAR file")
	baseline := fs.String("baseline", os.Getenv("BASELINE_DIR"), "Baseline dir for visual diff")
	stepsJSON := fs.String("steps", "", "JSON array of steps [{\"action\":\"click\",\"target\":\"text=...\"}]")
	fs.Parse(args)

	var blocked []string
	if env := os.Getenv("BLOCKED_HOSTS"); env != "" {
		for _, h := range strings.Split(env, ",") {
			if trimmed := strings.TrimSpace(h); trimmed != "" {
				blocked = append(blocked, trimmed)
			}
		}
	}
	var steps []runner.Step
	if strings.TrimSpace(*stepsJSON) != "" {
		if err := json.Unmarshal([]byte(*stepsJSON), &steps); err != nil {
			log.Fatalf("invalid steps JSON: %v", err)
		}
	}

	opts := runner.Options{
		TargetURL:    *url,
		ScriptPath:   *script,
		Engine:       *engine,
		ExtensionDir: strings.TrimSpace(*ext),
		Headless:     *headless,
		CaptureTrace: *trace,
		CaptureHAR:   *har,
		ReplayHAR:    strings.TrimSpace(*replayHar),
		BaselineDir:  strings.TrimSpace(*baseline),
		BlockedHosts: blocked,
		Steps:        steps,
		Workspace:    ".",
	}
	res, err := runner.Run(opts)
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}
	b, _ := json.MarshalIndent(res.Manifest, "", "  ")
	fmt.Println(string(b))
}

func listCmd() {
	runs, err := runner.FindRuns(".")
	if err != nil {
		log.Fatal(err)
	}
	for _, id := range runs {
		fmt.Println(id)
	}
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8787, "Port to listen on")
	fs.Parse(args)

	s := newServer(".")
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("lab serve listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s.routes()))
}

// --- auth middleware ---

type authMiddleware struct {
	apiKey string
}

func newAuthMiddleware() *authMiddleware {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Println("WARNING: No API_KEY set, authentication disabled (dev mode only)")
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
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid Authorization format (use: Bearer <key>)"})
			return
		}

		token := strings.TrimPrefix(auth, prefix)

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(token), []byte(a.apiKey)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
			return
		}

		next(w, r)
	}
}

// --- server ---

type server struct {
	workspace string
}

func newServer(workspace string) *server {
	return &server{workspace: workspace}
}

func (s *server) routes() http.Handler {
	auth := newAuthMiddleware()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/v1/runs", auth.authenticate(s.handleRuns))
	mux.HandleFunc("/v1/runs/", s.handleRunByID) // Read-only, no auth required
	mux.HandleFunc("/v1/extensions", auth.authenticate(s.handleExtensions))
	// static files for artifacts
	runsDir := filepath.Join(s.workspace, "runs")
	mux.Handle("/runs/", http.StripPrefix("/runs/", http.FileServer(http.Dir(runsDir))))

	// serve web UI
	uiDir := filepath.Join(s.workspace, "webui")
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir(uiDir))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	return withCORS(mux)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

type runRequest struct {
	URL             string        `json:"url"`
	Script          string        `json:"script"`
	ScriptURL       string        `json:"script_url"`
	ScriptGitRepo   string        `json:"script_git_repo"`
	ScriptGitPath   string        `json:"script_git_path"`
	Engine          string        `json:"engine"`
	ExtensionDir    string        `json:"extension_dir"`
	Headless        *bool         `json:"headless"`
	HAR             bool          `json:"har"`
	ReplayHAR       string        `json:"replay_har"`
	Baseline        string        `json:"baseline"`
	BlockedHosts    []string      `json:"blocked_hosts"`
	VisualThreshold float64       `json:"visual_threshold"`
	Steps           []runner.Step `json:"steps"`
}

func (s *server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var blocked []string
	if env := os.Getenv("BLOCKED_HOSTS"); env != "" {
		for _, h := range strings.Split(env, ",") {
			if trimmed := strings.TrimSpace(h); trimmed != "" {
				blocked = append(blocked, trimmed)
			}
		}
	}
	opts := runner.Options{
		TargetURL:           req.URL,
		ScriptPath:          req.Script,
		ScriptURL:           req.ScriptURL,
		ScriptGitRepo:       req.ScriptGitRepo,
		ScriptGitPath:       req.ScriptGitPath,
		Engine:              req.Engine,
		ExtensionDir:        strings.TrimSpace(req.ExtensionDir),
		Headless:            true,
		CaptureHAR:          req.HAR,
		ReplayHAR:           strings.TrimSpace(req.ReplayHAR),
		BaselineDir:         strings.TrimSpace(req.Baseline),
		VisualDiffThreshold: req.VisualThreshold,
		BlockedHosts:        blocked,
		Steps:               req.Steps,
		Workspace:           s.workspace,
	}
	if req.Headless != nil {
		opts.Headless = *req.Headless
	}

	res, err := runner.Run(opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	manifest := normalizeManifestPaths(res)
	writeJSON(w, http.StatusOK, manifest)
}

func (s *server) handleRunByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/runs/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id"})
		return
	}
	runID := parts[0]
	if len(parts) == 1 {
		manifestPath := filepath.Join(s.workspace, "runs", runID, "run.json")
		if _, err := os.Stat(manifestPath); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		manifest, err := runner.LoadManifest(manifestPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, normalizeManifestPaths(runner.Result{RunID: runID, Manifest: manifest}))
		return
	}

	if len(parts) >= 2 && parts[1] == "logs" {
		logPath := filepath.Join(s.workspace, "runs", runID, "logs", "runner.ndjson")
		http.ServeFile(w, r, logPath)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown path"})
}

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
		return "", fmt.Errorf("invalid extension: %s (allowed: .crx, .xpi, .zip)", ext)
	}

	// Generate safe name: hash + original extension
	hash := sha256.Sum256([]byte(base + time.Now().String()))
	safeName := hex.EncodeToString(hash[:8]) + ext

	return safeName, nil
}

func (s *server) handleExtensions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	// Sanitize filename - SECURITY FIX
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

	out, err := os.Create(dest)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	log.Printf("Extension uploaded: %s (original: %s)", safeName, header.Filename)
	writeJSON(w, http.StatusOK, map[string]string{"path": dest, "filename": safeName})
}

func normalizeManifestPaths(res runner.Result) runner.Manifest {
	m := res.Manifest
	prefix := "/runs/" + res.RunID + "/artifacts/"
	if m.Screenshot != "" && !strings.HasPrefix(m.Screenshot, "/runs/") {
		m.Screenshot = prefix + m.Screenshot
	}
	if m.VideoWebP != "" && !strings.HasPrefix(m.VideoWebP, "/runs/") {
		m.VideoWebP = prefix + m.VideoWebP
	}
	if m.VideoWebM != "" && !strings.HasPrefix(m.VideoWebM, "/runs/") {
		m.VideoWebM = prefix + m.VideoWebM
	}
	if m.HAR != "" && !strings.HasPrefix(m.HAR, "/runs/") {
		m.HAR = prefix + m.HAR
	}
	if m.TraceZip != "" && !strings.HasPrefix(m.TraceZip, "/runs/") {
		m.TraceZip = prefix + m.TraceZip
	}
	if m.VisualDiffImg != "" && !strings.HasPrefix(m.VisualDiffImg, "/runs/") {
		m.VisualDiffImg = prefix + m.VisualDiffImg
	}
	return m
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
