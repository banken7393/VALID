// Package cli defines the VALID Cobra command tree (plumbing only).
// Cycle UX is skills (plan/build/finish/…); this package is deterministic scaffolding.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var rootRepo string

// NewRoot builds the root cobra command with plumbing subcommands.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "valid",
		Short:         "Visual Agentic Loop for Isolated Development",
		Long:          "VALID CLI is deterministic plumbing for boards, worktrees, Dev Containers, gates, promote, merge, MCP, dashboard, and IDE install. Method UX is skills (plan/spec/build-feature/audit/finish/delegate/…), not these command names.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&rootRepo, "repo", "", "Path to the git repository root (default: current directory)")

	root.AddCommand(newInitCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newFeatureCmd())
	root.AddCommand(newEnvCmd())
	root.AddCommand(newBoardCmd())
	root.AddCommand(newTaskCmd())
	root.AddCommand(newReportCmd())
	root.AddCommand(newGateCmd())
	root.AddCommand(newPromoteCmd())
	root.AddCommand(newMergeCmd())
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newMCPCmd())

	return root
}

// Execute runs the root command.
func Execute() {
	if err := NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveRepo() (string, error) {
	start := rootRepo
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		start = wd
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if root, ok := findGitToplevel(abs); ok {
		return root, nil
	}
	// Not a git repo — allow init / install in a plain directory.
	return abs, nil
}

func findGitToplevel(start string) (string, bool) {
	dir := start
	for {
		gitPath := filepath.Join(dir, ".git")
		if st, err := os.Stat(gitPath); err == nil {
			// Directory or file (worktree) both count — prefer git rev-parse when available.
			if root, err := gitRevParseToplevel(dir); err == nil {
				return root, true
			}
			if st.IsDir() || st.Mode().IsRegular() {
				return dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func gitRevParseToplevel(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
