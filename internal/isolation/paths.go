// Package isolation resolves feature path contracts and detects principal-tree
// contamination (code edited outside the feature worktree while still using the
// principal folder as the agent control plane).
package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/banken7393/valid/internal/schema"
)

// FeaturePaths is the path contract for one feature (principal = control plane).
type FeaturePaths struct {
	Slug       string `json:"slug"`
	RepoRoot   string `json:"repo_root"`
	CodeRoot   string `json:"code_root"` // feature worktree abs path (empty for minipatch)
	Board      string `json:"board"`     // data.json abs
	HowItWorks string `json:"how_it_works"`
	Workspace  string `json:"workspace"`
	Branch     string `json:"branch"`
	BaseBranch string `json:"base_branch"`
	Mode       string `json:"mode"`
}

// Resolve returns absolute paths for the feature. CodeRoot is empty for minipatch
// (or when no worktree was created).
func Resolve(repoRoot string, b *schema.Board) (FeaturePaths, error) {
	if b == nil {
		return FeaturePaths{}, fmt.Errorf("board is nil")
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return FeaturePaths{}, err
	}
	p := FeaturePaths{
		Slug:       b.Feature,
		RepoRoot:   absRoot,
		Mode:       b.Mode,
		Branch:     b.Environment.Branch,
		BaseBranch: b.Environment.BaseBranch,
		Board:      schema.BoardPath(absRoot, b.Feature),
	}
	if b.Paths.HowItWorks != "" {
		if hp, err := schema.ResolveUnderRoot(absRoot, b.Paths.HowItWorks); err == nil {
			p.HowItWorks = hp
		}
	} else {
		p.HowItWorks = filepath.Join(schema.FeatureDir(absRoot, b.Feature), "how-it-works.mmd")
	}
	if b.Paths.WorkspaceDir != "" {
		if wp, err := schema.ResolveUnderRoot(absRoot, b.Paths.WorkspaceDir); err == nil {
			p.Workspace = wp
		}
	} else {
		p.Workspace = filepath.Join(schema.FeatureDir(absRoot, b.Feature), "workspace")
	}

	code := strings.TrimSpace(b.Environment.WorktreePath)
	if code == "" && b.Paths.Worktree != "" {
		if wp, err := schema.ResolveUnderRoot(absRoot, b.Paths.Worktree); err == nil {
			code = wp
		}
	}
	if code != "" {
		if !filepath.IsAbs(code) {
			code = filepath.Join(absRoot, code)
		}
		code, err = filepath.Abs(code)
		if err != nil {
			return FeaturePaths{}, err
		}
		p.CodeRoot = code
	}
	return p, nil
}

// FormatEnv prints shell-friendly KEY=value lines for agents/skills.
func (p FeaturePaths) FormatEnv() string {
	var b strings.Builder
	fmt.Fprintf(&b, "slug=%s\n", p.Slug)
	fmt.Fprintf(&b, "mode=%s\n", p.Mode)
	fmt.Fprintf(&b, "REPO_ROOT=%s\n", p.RepoRoot)
	fmt.Fprintf(&b, "CODE_ROOT=%s\n", nullPath(p.CodeRoot))
	fmt.Fprintf(&b, "BOARD=%s\n", p.Board)
	fmt.Fprintf(&b, "HOW_IT_WORKS=%s\n", nullPath(p.HowItWorks))
	fmt.Fprintf(&b, "WORKSPACE=%s\n", nullPath(p.Workspace))
	fmt.Fprintf(&b, "BRANCH=%s\n", nullStr(p.Branch))
	fmt.Fprintf(&b, "BASE_BRANCH=%s\n", nullStr(p.BaseBranch))
	return b.String()
}

func nullPath(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return s
}

func nullStr(s string) string {
	return strings.TrimSpace(s)
}

// CodeRootExists reports whether the feature worktree directory is present.
func (p FeaturePaths) CodeRootExists() bool {
	if p.CodeRoot == "" {
		return false
	}
	st, err := os.Stat(p.CodeRoot)
	return err == nil && st.IsDir()
}
