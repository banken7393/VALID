package cli

import (
	"fmt"
	"path/filepath"

	"github.com/banken7393/valid/internal/env"
	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/gate"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newMergeCmd() *cobra.Command {
	var skipGate bool
	var allowEarly bool
	var allowCurrentHead bool
	cmd := &cobra.Command{
		Use:   "merge <slug>",
		Short: "Merge feature branch into the principal folder's base branch",
		Long:  "Merges into the branch recorded when the worktree was created (principal HEAD at plan time), not a hard-coded main. Tears down the feature container before removing the worktree. Prefer skill finish.",
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
				return fmt.Errorf("minipatch has no feature branch to merge; commit on the current branch")
			}
			life := b.Lifecycle
			if !allowEarly && life != schema.LifeAudit && life != schema.LifeFinish && life != schema.LifeDone {
				return fmt.Errorf("refuse merge: lifecycle is %q (want audit|finish); use --allow-early to override", life)
			}
			base := b.Environment.BaseBranch
			if base == "" && !allowCurrentHead {
				return fmt.Errorf("refuse merge: environment.base_branch is empty (record it at feature create); use --allow-current-head to merge into principal HEAD")
			}
			if skipGate {
				fmt.Fprintln(cmd.OutOrStdout(), "WARNING: --skip-gate set; merging without AC/test gate")
			}
			if !skipGate {
				cfg, err := schema.LoadConfig(repo)
				if err != nil {
					return err
				}
				res := gate.Run(repo, b, cfg)
				_ = feature.Save(repo, b)
				if !res.Passed {
					return fmt.Errorf("refuse merge: gate failed for %q", slug)
				}
			}
			wt := b.Environment.WorktreePath
			if wt == "" {
				wt = filepath.Join(repo, env.WorktreesDir, slug)
			}
			mgr, err := env.NewManager(repo)
			if err != nil {
				return err
			}
			// Merge first so a failed merge leaves the quarantine container intact for recovery.
			if err := mgr.MergeWorktree(slug, base); err != nil {
				return err
			}
			if err := env.Down(wt, slug); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: env down: %v\n", err)
			}
			_ = b.SetLifecycle(schema.LifeDone)
			b.Environment.WorktreePath = ""
			b.Environment.DevcontainerPath = ""
			if err := feature.Save(repo, b); err != nil {
				return err
			}
			if base == "" {
				base = "(principal HEAD)"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Merged feature %q into %s\n", slug, base)
			return nil
		},
	}
	cmd.Flags().BoolVar(&skipGate, "skip-gate", false, "Skip gate check (dangerous)")
	cmd.Flags().BoolVar(&allowEarly, "allow-early", false, "Allow merge before lifecycle audit/finish")
	cmd.Flags().BoolVar(&allowCurrentHead, "allow-current-head", false, "Allow merge when base_branch is unset (uses principal HEAD)")
	return cmd
}
