package cli

import (
	"fmt"

	"github.com/banken7393/valid/internal/env"
	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newFeatureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feature",
		Short: "Feature board / worktree plumbing",
	}
	cmd.AddCommand(newFeatureNewCmd())
	cmd.AddCommand(newFeatureDeleteCmd())
	return cmd
}

func newFeatureNewCmd() *cobra.Command {
	var mode string
	var north string
	var noEnv bool

	cmd := &cobra.Command{
		Use:   "new <slug>",
		Short: "Create feature board (+ worktree/DC unless minipatch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			slug := args[0]
			if mode == "" {
				mode = schema.ModeFeature
			}
			opts := feature.Options{
				Mode:             mode,
				NorthStar:        north,
				SkipWorktree:     noEnv || mode == schema.ModeMinipatch,
				SkipDevcontainer: noEnv || mode == schema.ModeMinipatch,
			}
			b, err := feature.Create(repo, slug, opts)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created feature %q mode=%s board=%s\n", b.Feature, b.Mode, schema.BoardPath(repo, slug))
			if b.Environment.IsolationWarning {
				fmt.Fprintln(cmd.OutOrStdout(), "warning: isolation soft — no Dev Environment (or env up failed)")
			}
			if b.Environment.WorktreePath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "worktree: %s branch=%s\n", b.Environment.WorktreePath, b.Environment.Branch)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", schema.ModeFeature, "feature|patch|minipatch")
	cmd.Flags().StringVar(&north, "north-star", "", "Optional north star text")
	cmd.Flags().BoolVar(&noEnv, "no-env", false, "Skip worktree and Devcontainer")
	return cmd
}

func newFeatureDeleteCmd() *cobra.Command {
	var keepEnv bool
	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete feature board and tear down worktree/DC (does not merge code)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			slug := args[0]
			if !keepEnv {
				// Best-effort teardown before board delete.
				if b, err := feature.Load(repo, slug); err == nil && b.Environment.WorktreePath != "" {
					_ = env.Down(b.Environment.WorktreePath, slug)
				}
				if mgr, err := env.NewManager(repo); err == nil {
					_ = mgr.RemoveWorktree(slug, true)
				}
			}
			if err := feature.DeleteBoard(repo, slug); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted board for %q\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&keepEnv, "keep-env", false, "Do not tear down worktree/Dev Container")
	return cmd
}
