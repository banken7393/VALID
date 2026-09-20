package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/banken7393/valid/internal/env"
	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Feature Dev Environment lifecycle",
	}
	cmd.AddCommand(newEnvUpCmd())
	cmd.AddCommand(newEnvDownCmd())
	return cmd
}

func newEnvUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up <slug>",
		Short: "Start / ensure feature Devcontainer (soft failure → isolation warning)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			slug := args[0]
			b, err := feature.Load(repo, slug)
			if err != nil {
				return err
			}
			if b.Mode == schema.ModeMinipatch {
				fmt.Fprintln(cmd.OutOrStdout(), "minipatch has no Dev Environment; isolation warning expected")
				b.Environment.IsolationWarning = true
				return feature.Save(repo, b)
			}
			mgr, err := env.NewManager(repo)
			if err != nil {
				return err
			}
			wt := b.Environment.WorktreePath
			if wt == "" {
				wt, base, err := mgr.CreateWorktree(slug)
				if err != nil {
					b.Environment.IsolationWarning = true
					_ = feature.Save(repo, b)
					fmt.Fprintf(cmd.OutOrStdout(), "warning: could not create worktree: %v\n", err)
					return nil
				}
				b.Environment.WorktreePath = wt
				b.Environment.Branch = env.BranchName(slug)
				b.Environment.BaseBranch = base
			}
			cfg, _ := schema.LoadConfig(repo)
			mainAbs := ""
			mainRel := cfg.MainDevcontainer
			if mainRel == "" {
				mainRel = ".devcontainer/devcontainer.json"
			}
			candidate := mainRel
			if !filepath.IsAbs(candidate) {
				candidate = filepath.Join(repo, mainRel)
			}
			if _, err := os.Stat(candidate); err == nil {
				mainAbs = candidate
			}
			dc, err := env.InjectDevcontainer(wt, slug, cfg.ResolveFeatureDashboardPort(slug), mainAbs, repo)
			if err != nil {
				return err
			}
			b.Environment.DevcontainerPath = dc
			if err := env.Up(wt); err != nil {
				b.Environment.IsolationWarning = true
				_ = feature.Save(repo, b)
				fmt.Fprintf(cmd.OutOrStdout(), "warning: env up soft-failed: %v\n", err)
				return nil
			}
			b.Environment.IsolationWarning = false
			if err := feature.Save(repo, b); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Env up for %q at %s\n", slug, wt)
			return nil
		},
	}
}

func newEnvDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down <slug>",
		Short: "Tear down feature Devcontainer and remove worktree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			slug := args[0]
			b, err := feature.Load(repo, slug)
			if err != nil {
				// Board may already be deleted after finish; still try container + worktree cleanup.
				_ = env.Down(filepath.Join(repo, env.WorktreesDir, slug), slug)
				mgr, mErr := env.NewManager(repo)
				if mErr != nil {
					return mErr
				}
				_ = mgr.RemoveWorktree(slug, true)
				fmt.Fprintf(cmd.OutOrStdout(), "Cleaned worktree for %q (board missing)\n", slug)
				return nil
			}
			wt := b.Environment.WorktreePath
			if wt == "" {
				wt = filepath.Join(repo, env.WorktreesDir, slug)
			}
			if err := env.Down(wt, slug); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: env down container: %v\n", err)
			}
			mgr, err := env.NewManager(repo)
			if err != nil {
				return err
			}
			if err := mgr.RemoveWorktree(slug, true); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: worktree remove: %v\n", err)
			}
			b.Environment.WorktreePath = ""
			b.Environment.DevcontainerPath = ""
			if err := feature.Save(repo, b); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Env down for %q\n", slug)
			return nil
		},
	}
}
