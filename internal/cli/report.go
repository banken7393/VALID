package cli

import (
	"fmt"

	"github.com/banken7393/valid/internal/feature"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var (
		status   string
		testName string
		output   string
		phase    string
	)
	cmd := &cobra.Command{
		Use:   "report <slug>",
		Short: "Optional helper: record a TDD result (prefer LLM file edits; gate runs test_command)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			b, err := feature.Load(repo, args[0])
			if err != nil {
				return err
			}
			if err := b.ApplyReport(status, testName, output, phase); err != nil {
				return err
			}
			if b.Lifecycle == "plan" || b.Lifecycle == "spec" {
				_ = b.SetLifecycle("build")
			}
			if err := feature.Save(repo, b); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Reported %s → %s (tdd phase=%s total=%d failed=%d)\n",
				testName, status, b.TDD.Phase, b.TDD.Total, b.TDD.Failed)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "pass", "pass|fail|pending")
	cmd.Flags().StringVar(&testName, "test-name", "", "Test identifier (required)")
	cmd.Flags().StringVar(&output, "output", "", "Optional output snippet")
	cmd.Flags().StringVar(&phase, "phase", "", "Optional tdd phase red|green|refactor")
	_ = cmd.MarkFlagRequired("test-name")
	return cmd
}
