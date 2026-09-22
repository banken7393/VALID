// Package dashboard serves the embedded VALID visual board over HTTP.
package dashboard

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/banken7393/valid/internal/schema"
	"github.com/banken7393/valid/web"
)

// Server serves the dashboard UI and board API.
type Server struct {
	DataPath string
	RepoRoot string
	Addr     string
	Theme    string // dark (default) | light — from .valid/config.json
}

// New creates a dashboard server for a board JSON path.
func New(dataPath, addr string) *Server {
	return &Server{DataPath: dataPath, Addr: addr, Theme: schema.ThemeDark}
}

// NewWithRepo creates a dashboard that can resolve how-it-works from paths on disk.
func NewWithRepo(dataPath, repoRoot, addr string) *Server {
	return &Server{DataPath: dataPath, RepoRoot: repoRoot, Addr: addr, Theme: schema.ThemeDark}
}

// Handler returns the HTTP mux for the dashboard.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	static, err := fs.Sub(web.FS, ".")
	if err != nil {
		static = web.FS
	}
	fileServer := http.FileServer(http.FS(static))

	mux.HandleFunc("/api/data", s.handleData)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"theme": s.resolvedTheme(),
		})
	})
	mux.Handle("/", fileServer)
	return mux
}

func (s *Server) resolvedTheme() string {
	return schema.NormalizeTheme(s.Theme)
}

func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Prefer live config so theme edits apply without restarting the process.
	theme := s.resolvedTheme()
	if s.RepoRoot != "" {
		if cfg, err := schema.LoadConfig(s.RepoRoot); err == nil {
			theme = schema.NormalizeTheme(cfg.Theme)
			s.Theme = theme
		}
	}
	b, err := schema.LoadBoard(s.DataPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error(), "theme": theme})
		return
	}
	enrichHowItWorks(s.RepoRoot, b)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"theme": theme,
		"board": b,
	})
}

func enrichHowItWorks(repoRoot string, b *schema.Board) {
	if b == nil || repoRoot == "" || strings.TrimSpace(b.Paths.HowItWorks) == "" {
		return
	}
	abs, err := schema.ResolveUnderRoot(repoRoot, b.Paths.HowItWorks)
	if err != nil {
		return
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return
	}
	// File on disk is the live source — prefer it over a stale inline board field
	// so dashboard polls pick up agent edits to how-it-works.mmd.
	if text := strings.TrimSpace(string(raw)); text != "" {
		b.HowItWorks = text
	}
}

// ListenAndServe starts the HTTP server until it fails.
func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:              s.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	fmt.Printf("VALID dashboard listening on http://%s (board: %s theme: %s)\n", displayAddr(s.Addr), s.DataPath, s.resolvedTheme())
	return srv.ListenAndServe()
}

func displayAddr(addr string) string {
	if len(addr) > 0 && addr[0] == ':' {
		return "127.0.0.1" + addr
	}
	return addr
}
