// Package env wraps git worktree and Dev Containers CLI operations.
package env

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// ValidDir is the project-local VALID runtime directory name.
	ValidDir = ".valid"
	// WorktreesDir is the relative path for feature worktrees.
	WorktreesDir = ".valid/worktrees"
)

var featureNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Manager performs worktree and devcontainer operations against a git repo root.
type Manager struct {
	RepoRoot string
}

// NewManager validates and returns a Manager for repoRoot.
func NewManager(repoRoot string) (*Manager, error) {
	if repoRoot == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}
	if !isGitRepo(abs) {
		return nil, fmt.Errorf("not a git repository: %s", abs)
	}
	return &Manager{RepoRoot: abs}, nil
}

func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// ValidateFeatureName ensures the feature name is filesystem-safe.
func ValidateFeatureName(name string) error {
	if name == "" {
		return fmt.Errorf("feature name is required")
	}
	if !featureNamePattern.MatchString(name) {
		return fmt.Errorf("invalid feature name %q (use letters, numbers, ._-)", name)
	}
	return nil
}

// BranchName returns the ephemeral feature branch name feature/<feature>.
func BranchName(feature string) string {
	return "feature/" + feature
}

// WorktreePath returns the absolute worktree path for a feature.
func (m *Manager) WorktreePath(feature string) string {
	return filepath.Join(m.RepoRoot, WorktreesDir, feature)
}

// ResolveBaseBranch returns the branch currently checked out in the principal repo folder.
// Worktrees are cut from this branch and merge returns into it — not a hard-coded main.
func (m *Manager) ResolveBaseBranch() (string, error) {
	cur, err := m.CurrentBranch()
	if err != nil {
		return "", err
	}
	if cur == "" || cur == "HEAD" {
		return "", fmt.Errorf("principal repo has detached HEAD; check out a branch before creating a worktree")
	}
	if strings.HasPrefix(cur, "feature/") {
		return "", fmt.Errorf("principal repo is on %q; check out your integration branch (the one in the main folder) before creating a worktree", cur)
	}
	return cur, nil
}

func (m *Manager) branchExists(name string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	cmd.Dir = m.RepoRoot
	return cmd.Run() == nil
}

// CreateWorktree adds a new git worktree for the feature on branch feature/<feature>
// starting from the principal folder's current branch. Returns worktree path and that base branch.
func (m *Manager) CreateWorktree(feature string) (worktreePath, baseBranch string, err error) {
	if err := ValidateFeatureName(feature); err != nil {
		return "", "", err
	}
	wt := m.WorktreePath(feature)
	if err := os.MkdirAll(filepath.Dir(wt), 0o755); err != nil {
		return "", "", fmt.Errorf("create worktrees dir: %w", err)
	}
	if _, err := os.Stat(wt); err == nil {
		return "", "", fmt.Errorf("worktree already exists: %s", wt)
	}

	base, err := m.ResolveBaseBranch()
	if err != nil {
		return "", "", err
	}

	branch := BranchName(feature)
	cmd := exec.Command("git", "worktree", "add", "-b", branch, wt, base)
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(wt)
		return "", "", fmt.Errorf("git worktree add: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	if err := m.PrepareWorktree(feature, wt); err != nil {
		_ = m.RemoveWorktree(feature, true)
		return "", "", err
	}
	return wt, base, nil
}

// RemoveWorktree removes the worktree and optionally deletes the branch.
func (m *Manager) RemoveWorktree(feature string, deleteBranch bool) error {
	if err := ValidateFeatureName(feature); err != nil {
		return err
	}
	wt := m.WorktreePath(feature)
	cmd := exec.Command("git", "worktree", "remove", "--force", wt)
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(wt)
		if _, statErr := os.Stat(wt); statErr == nil {
			return fmt.Errorf("git worktree remove: %v\n%s", err, strings.TrimSpace(string(out)))
		}
	}
	if deleteBranch {
		branch := BranchName(feature)
		del := exec.Command("git", "branch", "-D", branch)
		del.Dir = m.RepoRoot
		_ = del.Run()
	}
	return nil
}

// MergeWorktree checks out baseBranch in the principal folder, merges feature/<feature>, then removes the worktree.
// baseBranch should be the branch recorded when the worktree was created; if empty, uses the principal HEAD.
func (m *Manager) MergeWorktree(feature, baseBranch string) error {
	if err := ValidateFeatureName(feature); err != nil {
		return err
	}
	base := strings.TrimSpace(baseBranch)
	if base == "" {
		var err error
		base, err = m.ResolveBaseBranch()
		if err != nil {
			return err
		}
	}
	if !m.branchExists(base) {
		return fmt.Errorf("base branch %q does not exist in principal repo", base)
	}
	branch := BranchName(feature)

	checkout := exec.Command("git", "checkout", base)
	checkout.Dir = m.RepoRoot
	if out, err := checkout.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout %s: %v\n%s", base, err, strings.TrimSpace(string(out)))
	}

	cmd := exec.Command("git", "merge", "--no-ff", "-m", fmt.Sprintf("Merge feature %s via VALID", feature), branch)
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge into %s: %v\n%s", base, err, strings.TrimSpace(string(out)))
	}
	return m.RemoveWorktree(feature, true)
}

// CurrentBranch returns the current branch name in the principal repo.
func (m *Manager) CurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = m.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
