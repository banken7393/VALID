package cli

import (
	"fmt"

	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/gate"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newGateCmd() *cobra.Command {
	var trustBoard bool
	cmd := &cobra.Command{
		Use:   "gate <slug>",
		Short: "Run dual gates (AC↔task coverage + executable tests)",
		Long:  "By default executes config.test_command in the feature worktree (or repo for minipatch) and records results on the board. Use --trust-board to skip execution and only inspect existing tdd evidence.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			cfg, err := schema.LoadConfig(repo)
			if err != nil {
				return err
			}
			b, err := feature.Load(repo, args[0])
			if err != nil {
				return err
			}
			_ = b.SetLifecycle(schema.LifeAudit)
			res := gate.RunWithOptions(repo, b, cfg, gate.Options{TrustBoard: trustBoard})
			if err := feature.Save(repo, b); err != nil {
				return err
			}
			for _, f := range res.Findings {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", f.Severity, f.Code, f.Message)
			}
			if !res.Passed {
				return fmt.Errorf("gate failed for %q", args[0])
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Gate passed for %q\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&trustBoard, "trust-board", false, "Skip running test_command; only inspect board tdd evidence")
	return cmd
}
