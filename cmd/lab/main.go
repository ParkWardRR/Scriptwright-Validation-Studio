package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"philadelphia/internal/runner"
	"strings"
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
	fs.Parse(args)

	var blocked []string
	if env := os.Getenv("BLOCKED_HOSTS"); env != "" {
		for _, h := range strings.Split(env, ",") {
			if trimmed := strings.TrimSpace(h); trimmed != "" {
				blocked = append(blocked, trimmed)
			}
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

// --- server ---

type server struct {
	workspace string
}

func newServer(workspace string) *server {
	return &server{workspace: workspace}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/v1/runs", s.handleRuns)
	mux.HandleFunc("/v1/runs/", s.handleRunByID)
	// static files for artifacts
	runsDir := filepath.Join(s.workspace, "runs")
	mux.Handle("/runs/", http.StripPrefix("/runs/", http.FileServer(http.Dir(runsDir))))
	return withCORS(mux)
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

type runRequest struct {
	URL          string `json:"url"`
	Script       string `json:"script"`
	Engine       string `json:"engine"`
	ExtensionDir string `json:"extension_dir"`
	Headless     *bool  `json:"headless"`
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
	opts := runner.Options{
		TargetURL:    req.URL,
		ScriptPath:   req.Script,
		Engine:       req.Engine,
		ExtensionDir: strings.TrimSpace(req.ExtensionDir),
		Headless:     true,
		Workspace:    s.workspace,
	}
	if req.Headless != nil {
		opts.Headless = *req.Headless
	}

	res, err := runner.Run(opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	manifest := res.Manifest
	manifest.Screenshot = "/runs/" + res.RunID + "/artifacts/" + manifest.Screenshot
	if manifest.VideoWebP != "" {
		manifest.VideoWebP = "/runs/" + res.RunID + "/artifacts/" + manifest.VideoWebP
	}
	if manifest.VideoWebM != "" {
		manifest.VideoWebM = "/runs/" + res.RunID + "/artifacts/" + manifest.VideoWebM
	}
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
		manifest.Screenshot = "/runs/" + runID + "/artifacts/" + manifest.Screenshot
		if manifest.VideoWebP != "" {
			manifest.VideoWebP = "/runs/" + runID + "/artifacts/" + manifest.VideoWebP
		}
		if manifest.VideoWebM != "" {
			manifest.VideoWebM = "/runs/" + runID + "/artifacts/" + manifest.VideoWebM
		}
		writeJSON(w, http.StatusOK, manifest)
		return
	}

	if len(parts) >= 2 && parts[1] == "logs" {
		logPath := filepath.Join(s.workspace, "runs", runID, "logs", "runner.ndjson")
		http.ServeFile(w, r, logPath)
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown path"})
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
