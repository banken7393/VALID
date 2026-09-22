package isolation

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/banken7393/valid/internal/schema"
)

// Finding codes used by doctor / gate.
const (
	CodeOutsideWorktree = "code_outside_worktree"
	CodeWorktreeMissing = "worktree_missing"
	CodeIsolationWarn   = "isolation_warning"
)

// allowedPrincipalPrefixes are control-plane paths agents may touch on the
// principal tree while a feature is in progress (board, knowledge, IDE install).
var allowedPrincipalPrefixes = []string{
	".valid/",
	".cursor/",
	".codex/",
	".claude/",
	".opencode/",
	"packages/cursor/",
	"packages/claude-code/",
	"packages/codex/",
	"packages/opencode/",
}

var allowedPrincipalFiles = map[string]struct{}{
	".gitignore": {},
	"agents.md":  {},
	"claude.md":  {},
}

// DirtyPaths lists principal working-tree paths that look like product code
// edited outside the feature worktree.
func DirtyPaths(repoRoot string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain", "-uall")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Not a git repo / git missing — doctor cannot prove contamination.
		if _, ok := err.(*exec.ExitError); ok || strings.Contains(string(out), "not a git repository") {
			return nil, fmt.Errorf("git status failed in %s: %v\n%s", repoRoot, err, strings.TrimSpace(string(out)))
		}
		return nil, fmt.Errorf("git status: %w", err)
	}
	var dirty []string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 4 {
			continue
		}
		// porcelain v1: XY SPACE path  OR  XY SPACE old -> new
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		path = strings.Trim(path, "\"")
		path = filepath.ToSlash(path)
		if isAllowedPrincipalPath(path) {
			continue
		}
		dirty = append(dirty, path)
	}
	return dirty, nil
}

func isAllowedPrincipalPath(path string) bool {
	p := filepath.ToSlash(strings.TrimSpace(path))
	if p == "" {
		return true
	}
	lower := strings.ToLower(p)
	if _, ok := allowedPrincipalFiles[lower]; ok {
		return true
	}
	base := strings.ToLower(filepath.Base(p))
	if _, ok := allowedPrincipalFiles[base]; ok && !strings.Contains(p, "/") {
		return true
	}
	for _, pref := range allowedPrincipalPrefixes {
		if strings.HasPrefix(p, pref) || strings.HasPrefix(lower, strings.ToLower(pref)) {
			return true
		}
	}
	return false
}

// Inspect returns isolation findings for a board. Minipatch skips worktree checks.
// When cfg.Isolation.Strict is true, contamination and board isolation_warning
// become errors (except minipatch).
func Inspect(repoRoot string, b *schema.Board, cfg schema.Config) []schema.Finding {
	if b == nil {
		return nil
	}
	if b.Mode == schema.ModeMinipatch {
		return nil
	}

	var findings []schema.Finding
	paths, err := Resolve(repoRoot, b)
	if err != nil {
		findings = append(findings, schema.Finding{
			Code:     CodeWorktreeMissing,
			Severity: schema.SevError,
			Message:  "cannot resolve feature paths: " + err.Error(),
		})
		return findings
	}

	if paths.CodeRoot == "" {
		findings = append(findings, schema.Finding{
			Code:     CodeIsolationWarn,
			Severity: severity(cfg, schema.SevWarning),
			Message:  "no feature worktree path on the board — stay on minipatch or recreate with valid feature new",
		})
	} else if !paths.CodeRootExists() {
		findings = append(findings, schema.Finding{
			Code:     CodeWorktreeMissing,
			Severity: schema.SevError,
			Message:  "feature worktree path missing: " + paths.CodeRoot,
		})
	}

	dirty, derr := DirtyPaths(repoRoot)
	if derr != nil {
		// Soft: cannot run git — do not hard-fail doctor/gate on tooling gaps.
		findings = append(findings, schema.Finding{
			Code:     CodeIsolationWarn,
			Severity: schema.SevWarning,
			Message:  "could not scan principal git status for code_outside_worktree: " + derr.Error(),
		})
	} else if len(dirty) > 0 {
		maxShow := 8
		shown := dirty
		extra := ""
		if len(dirty) > maxShow {
			shown = dirty[:maxShow]
			extra = fmt.Sprintf(" (+%d more)", len(dirty)-maxShow)
		}
		findings = append(findings, schema.Finding{
			Code:     CodeOutsideWorktree,
			Severity: severity(cfg, schema.SevWarning),
			Message: fmt.Sprintf(
				"product files dirty on the principal tree (write feature code under CODE_ROOT instead): %s%s",
				strings.Join(shown, ", "),
				extra,
			),
		})
	}

	if b.Environment.IsolationWarning {
		findings = append(findings, schema.Finding{
			Code:     CodeIsolationWarn,
			Severity: severity(cfg, schema.SevWarning),
			Message:  "board environment.isolation_warning=true (soft isolation / env up failed / worked outside quarantine)",
		})
	}

	return findings
}

func severity(cfg schema.Config, soft string) string {
	if cfg.Isolation.Strict {
		return schema.SevError
	}
	return soft
}

// HasCodeOutside reports whether findings include code_outside_worktree.
func HasCodeOutside(findings []schema.Finding) bool {
	for _, f := range findings {
		if f.Code == CodeOutsideWorktree {
			return true
		}
	}
	return false
}
