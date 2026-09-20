// Package scripts loads project utility scripts exposed as MCP tools.
// Layout: .valid/scripts/<tool-id>/meta.json (+ optional run.sh / assets).
package scripts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	MetaFileName   = "meta.json"
	DefaultTimeout = 120 * time.Second
	MaxOutputBytes = 256 * 1024
)

var toolIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

// reserved MCP tool names (RAG + aliases) — scripts may not use these ids.
var reservedToolNames = map[string]struct{}{
	"search_knowledge": {},
	"list_documents":   {},
	"get_document":     {},
	"upsert_document":  {},
}

// Meta is the per-script contract the user edits.
type Meta struct {
	// Description is required — shown as the MCP tool description (model-facing).
	Description string `json:"description"`
	// Command is argv executed relative to the script directory (or Cwd).
	// Example: ["bash", "run.sh"]
	Command []string `json:"command"`
	// TimeoutSec overrides the default run timeout (seconds). 0 = default.
	TimeoutSec int `json:"timeout_sec,omitempty"`
	// Cwd is "script" (default: script folder) or "repo" (repository root).
	Cwd string `json:"cwd,omitempty"`
	// Enabled defaults to true when omitted. Set false to keep a script on disk without exposing it.
	Enabled *bool `json:"enabled,omitempty"`
}

// Script is a loaded, runnable MCP tool.
type Script struct {
	ID          string
	Description string
	Command     []string
	Timeout     time.Duration
	Cwd         string // absolute
	Dir         string // absolute script folder
	MetaPath    string
}

// Result is the outcome of Run.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
	Duration time.Duration
}

// LoadAll scans scriptsRoot for <id>/meta.json entries.
func LoadAll(repoRoot, scriptsRoot string) ([]Script, error) {
	if scriptsRoot == "" {
		return nil, nil
	}
	absScripts := scriptsRoot
	if !filepath.IsAbs(absScripts) {
		absScripts = filepath.Join(repoRoot, scriptsRoot)
	}
	entries, err := os.ReadDir(absScripts)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read scripts root: %w", err)
	}

	var out []Script
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if !toolIDPattern.MatchString(id) {
			continue
		}
		if _, reserved := reservedToolNames[id]; reserved {
			continue
		}
		dir := filepath.Join(absScripts, id)
		metaPath := filepath.Join(dir, MetaFileName)
		raw, err := os.ReadFile(metaPath)
		if err != nil {
			continue // no meta → not an MCP script
		}
		var meta Meta
		if err := json.Unmarshal(raw, &meta); err != nil {
			continue
		}
		if meta.Enabled != nil && !*meta.Enabled {
			continue
		}
		desc := strings.TrimSpace(meta.Description)
		if desc == "" || len(meta.Command) == 0 {
			continue
		}
		cmd := make([]string, 0, len(meta.Command))
		validCmd := true
		for _, part := range meta.Command {
			part = strings.TrimSpace(part)
			if part == "" {
				validCmd = false
				break
			}
			cmd = append(cmd, part)
		}
		if !validCmd || len(cmd) == 0 {
			continue
		}
		cwd := dir
		switch strings.ToLower(strings.TrimSpace(meta.Cwd)) {
		case "", "script":
			cwd = dir
		case "repo":
			cwd = repoRoot
		default:
			cwd = dir
		}
		timeout := DefaultTimeout
		if meta.TimeoutSec > 0 {
			timeout = time.Duration(meta.TimeoutSec) * time.Second
		}
		if timeout > 15*time.Minute {
			timeout = 15 * time.Minute
		}
		out = append(out, Script{
			ID:          id,
			Description: desc,
			Command:     cmd,
			Timeout:     timeout,
			Cwd:         cwd,
			Dir:         dir,
			MetaPath:    metaPath,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Find returns one script by id.
func Find(repoRoot, scriptsRoot, id string) (*Script, error) {
	all, err := LoadAll(repoRoot, scriptsRoot)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("script %q not found or disabled", id)
}

// Run executes the script with optional extra argv appended.
func (s *Script) Run(ctx context.Context, extraArgs []string) (*Result, error) {
	if s == nil {
		return nil, fmt.Errorf("script is nil")
	}
	if len(s.Command) == 0 {
		return nil, fmt.Errorf("script %q has empty command", s.ID)
	}

	argv := append([]string{}, s.Command...)
	argv = append(argv, extraArgs...)

	bin := argv[0]
	if !filepath.IsAbs(bin) {
		// Resolve relative binary against script dir (comfortable for ./run.sh).
		candidate := filepath.Join(s.Dir, bin)
		if _, err := os.Stat(candidate); err == nil {
			bin = candidate
		}
	}

	timeout := s.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, bin, argv[1:]...)
	cmd.Dir = s.Cwd
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &limitedWriter{buf: &stdout, limit: MaxOutputBytes}
	cmd.Stderr = &limitedWriter{buf: &stderr, limit: MaxOutputBytes}

	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start)

	res := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: dur,
		ExitCode: 0,
	}
	if runCtx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		return res, nil
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, fmt.Errorf("run %s: %w", s.ID, err)
	}
	return res, nil
}

// ToolDefinition returns an MCP tools/list entry for this script.
func (s *Script) ToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        s.ID,
		"description": s.Description,
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"args": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Optional extra argv appended to the script command",
				},
			},
		},
	}
}

// FormatResult is a compact text payload for MCP tools/call.
func FormatResult(s *Script, res *Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "script=%s exit=%d duration=%s timed_out=%v\n", s.ID, res.ExitCode, res.Duration.Round(time.Millisecond), res.TimedOut)
	if strings.TrimSpace(res.Stdout) != "" {
		b.WriteString("--- stdout ---\n")
		b.WriteString(res.Stdout)
		if !strings.HasSuffix(res.Stdout, "\n") {
			b.WriteByte('\n')
		}
	}
	if strings.TrimSpace(res.Stderr) != "" {
		b.WriteString("--- stderr ---\n")
		b.WriteString(res.Stderr)
		if !strings.HasSuffix(res.Stderr, "\n") {
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String())
}

type limitedWriter struct {
	buf   *bytes.Buffer
	limit int
	n     int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.n >= w.limit {
		return len(p), nil
	}
	remain := w.limit - w.n
	if len(p) > remain {
		_, _ = w.buf.Write(p[:remain])
		_, _ = w.buf.WriteString("\n…[output truncated]\n")
		w.n = w.limit
		return len(p), nil
	}
	n, err := w.buf.Write(p)
	w.n += n
	return len(p), err
}
