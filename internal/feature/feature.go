// Package feature manages per-feature board paths and bootstrap.
package feature

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/banken7393/valid/internal/env"
	"github.com/banken7393/valid/internal/schema"
)

var slugPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Options controls feature bootstrap.
type Options struct {
	Mode             string // feature|patch|minipatch
	SkipWorktree     bool
	SkipDevcontainer bool
	NorthStar        string
}

// ValidateSlug ensures a filesystem-safe feature slug.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("feature slug is required")
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("invalid feature slug %q", slug)
	}
	return nil
}

// Create bootstraps a feature board and optionally worktree + DC.
// On failure after creating a worktree, the worktree/branch are rolled back.
func Create(repoRoot, slug string, opts Options) (b *schema.Board, err error) {
	if err := ValidateSlug(slug); err != nil {
		return nil, err
	}
	mode := opts.Mode
	if mode == "" {
		mode = schema.ModeFeature
	}
	if mode == schema.ModeMinipatch {
		opts.SkipWorktree = true
		opts.SkipDevcontainer = true
	}

	cfg, err := schema.LoadConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	boardPath := schema.BoardPath(repoRoot, slug)
	if _, err := os.Stat(boardPath); err == nil {
		return nil, fmt.Errorf("feature board already exists: %s", boardPath)
	}

	featDir := schema.FeatureDir(repoRoot, slug)
	workspace := filepath.Join(featDir, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir feature: %w", err)
	}
	createdFeatDir := true
	defer func() {
		if err != nil && createdFeatDir {
			_ = os.RemoveAll(featDir)
		}
	}()

	howPath := filepath.Join(featDir, "how-it-works.mmd")
	howFileBody := "# Author how-it-works Mermaid during skill plan (flowchart/sequence). Stub diagrams fail the gate.\n"
	if err = os.WriteFile(howPath, []byte(howFileBody), 0o644); err != nil {
		return nil, fmt.Errorf("write how-it-works: %w", err)
	}

	depsPath := filepath.Join(workspace, "deps-delta.md")
	depsBody := "# Deps delta\n\nRecord host/deps changes validated in the feature env before promote env.\n"
	if err = os.WriteFile(depsPath, []byte(depsBody), 0o644); err != nil {
		return nil, fmt.Errorf("write deps-delta: %w", err)
	}

	b = schema.NewBoard(slug, mode)
	b.NorthStar = opts.NorthStar
	b.HowItWorks = ""
	b.Paths = schema.Paths{
		BoardDir:     filepath.ToSlash(filepath.Join(schema.ValidDir, schema.FeaturesDir, slug)),
		HowItWorks:   filepath.ToSlash(filepath.Join(schema.ValidDir, schema.FeaturesDir, slug, "how-it-works.mmd")),
		WorkspaceDir: filepath.ToSlash(filepath.Join(schema.ValidDir, schema.FeaturesDir, slug, "workspace")),
		DepsDelta:    filepath.ToSlash(filepath.Join(schema.ValidDir, schema.FeaturesDir, slug, "workspace", "deps-delta.md")),
	}
	b.Environment.DatabaseEnabled = cfg.Database.Enabled
	if cfg.Database.Enabled {
		recipeNote := "# Database isolation\n\n`database.enabled=true` — do **not** use the principal DSN.\n\n"
		if strings.TrimSpace(cfg.Database.Recipe) != "" {
			recipeNote += "## Recipe\n\n```\n" + cfg.Database.Recipe + "\n```\n"
		} else {
			recipeNote += "No `database.recipe` configured — document how to start an ephemeral DB for this feature.\n"
			b.Environment.IsolationWarning = true
		}
		_ = os.WriteFile(filepath.Join(workspace, "database.md"), []byte(recipeNote), 0o644)
	}

	mainAbs := ""
	mainRel := cfg.MainDevcontainer
	if mainRel == "" {
		mainRel = ".devcontainer/devcontainer.json"
	}
	candidate := mainRel
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(repoRoot, mainRel)
	}
	if _, statErr := os.Stat(candidate); statErr == nil {
		mainAbs = candidate
	}

	var mgr *env.Manager
	worktreeCreated := false
	defer func() {
		if err != nil && worktreeCreated && mgr != nil {
			_ = env.Down(b.Environment.WorktreePath, slug)
			_ = mgr.RemoveWorktree(slug, true)
		}
	}()

	if !opts.SkipWorktree {
		mgr, err = env.NewManager(repoRoot)
		if err != nil {
			return nil, err
		}
		var wt, base string
		wt, base, err = mgr.CreateWorktree(slug)
		if err != nil {
			return nil, err
		}
		worktreeCreated = true
		b.Environment.WorktreePath = wt
		b.Environment.Branch = env.BranchName(slug)
		b.Environment.BaseBranch = base
		b.Paths.Worktree = filepath.ToSlash(filepath.Join(schema.ValidDir, schema.WorktreesDir, slug))

		if !opts.SkipDevcontainer {
			featPort := cfg.ResolveFeatureDashboardPort(slug)
			var dcPath string
			dcPath, err = env.InjectDevcontainer(wt, slug, featPort, mainAbs, repoRoot)
			if err != nil {
				return nil, err
			}
			b.Environment.DevcontainerPath = dcPath
			if upErr := env.Up(wt); upErr != nil {
				b.Environment.IsolationWarning = true
			}
		}
	} else {
		b.Environment.IsolationWarning = true
	}

	if err = schema.SaveBoard(boardPath, b); err != nil {
		return nil, err
	}
	createdFeatDir = false
	worktreeCreated = false
	return b, nil
}

// Load loads a feature board by slug.
func Load(repoRoot, slug string) (*schema.Board, error) {
	return schema.LoadBoard(schema.BoardPath(repoRoot, slug))
}

// Save persists a feature board.
func Save(repoRoot string, b *schema.Board) error {
	if b == nil {
		return fmt.Errorf("board is nil")
	}
	return schema.SaveBoard(schema.BoardPath(repoRoot, b.Feature), b)
}

// DeleteBoard removes the feature board directory (not the worktree).
func DeleteBoard(repoRoot, slug string) error {
	dir := schema.FeatureDir(repoRoot, slug)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete feature board: %w", err)
	}
	return nil
}

// ResolveBoardPath returns the data.json path for a slug.
func ResolveBoardPath(repoRoot, slug string) string {
	return schema.BoardPath(repoRoot, slug)
}
