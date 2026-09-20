package mcp

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ServeHTTP starts an HTTP MCP server with SSE notifications and JSON-RPC POST.
// Security: bind should be loopback; requires HTTPToken (Authorization: Bearer).
// Project scripts are disabled on this transport unless ScriptsDisabled=false was set
// (CLI defaults scripts OFF for HTTP).
//
// Endpoints:
//   GET  /mcp  — open SSE stream, returns session id header
//   POST /mcp  — JSON-RPC request
func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	if strings.TrimSpace(s.httpToken) == "" {
		return fmt.Errorf("MCP HTTP requires a token (set VALID_MCP_TOKEN or pass --token); refusing to start unauthenticated HTTP")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// allow ":7433" form
		if strings.HasPrefix(addr, ":") {
			host = ""
		} else {
			return fmt.Errorf("invalid MCP HTTP addr %q: %w", addr, err)
		}
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return fmt.Errorf("MCP HTTP must bind to loopback (e.g. 127.0.0.1:7433), got %q", addr)
	}
	if ip := net.ParseIP(host); ip != nil {
		if !ip.IsLoopback() {
			return fmt.Errorf("MCP HTTP must bind to loopback, got %q", addr)
		}
	} else {
		// Non-IP hostnames: only localhost / *.localhost (DNS rebinding otherwise bypasses IP checks).
		h := strings.ToLower(host)
		if h != "localhost" && !strings.HasSuffix(h, ".localhost") {
			return fmt.Errorf("MCP HTTP host must be a loopback IP or localhost, got %q", host)
		}
	}

	sessions := &sseHub{conns: map[string]chan []byte{}}

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if !s.authorizeHTTP(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.handleSSE(w, r, sessions)
		case http.MethodPost:
			s.handleHTTPPost(w, r, sessions)
		case http.MethodOptions:
			w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func (s *Server) authorizeHTTP(r *http.Request) bool {
	want := strings.TrimSpace(s.httpToken)
	if want == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

type sseHub struct {
	mu    sync.RWMutex
	conns map[string]chan []byte
}

func (h *sseHub) add(id string) chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.conns[id] = ch
	h.mu.Unlock()
	return ch
}

func (h *sseHub) remove(id string) {
	h.mu.Lock()
	if ch, ok := h.conns[id]; ok {
		delete(h.conns, id)
		close(ch)
	}
	h.mu.Unlock()
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request, hub *sseHub) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	sessionID := newSessionID()
	ch := hub.add(sessionID)
	defer hub.remove(sessionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Mcp-Session-Id", sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, open := <-ch:
			if !open {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleHTTPPost(w http.ResponseWriter, r *http.Request, hub *sseHub) {
	_ = hub
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	resp, err := s.HandleBytes(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if resp == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}

// MustEncode is used by tests.
func MustEncode(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return b
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
