package cli

import (
	"fmt"
	"os"

	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/isolation"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newPathsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "paths <slug>",
		Short: "Print feature path contract (CODE_ROOT, BOARD, …)",
		Long:  "Principal repo stays the agent control plane; feature product code must be written under CODE_ROOT (.valid/worktrees/<slug>/). Board/Mermaid stay on the principal under .valid/features/<slug>/.",
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
			p, err := isolation.Resolve(repo, b)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), p.FormatEnv())
			if p.CodeRoot != "" && !p.CodeRootExists() {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: CODE_ROOT does not exist on disk: %s\n", p.CodeRoot)
			}
			return nil
		},
	}
	return cmd
}

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor <slug>",
		Short: "Check path contract and principal-tree contamination",
		Long:  "Scans the principal git working tree for product files dirty outside CODE_ROOT. With isolation.strict=true in config, contamination fails (exit 1). Always prints paths + findings.",
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
			p, err := isolation.Resolve(repo, b)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), p.FormatEnv())
			fmt.Fprintln(cmd.OutOrStdout())

			findings := isolation.Inspect(repo, b, cfg)
			if len(findings) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "doctor: ok")
				return nil
			}
			failed := false
			for _, f := range findings {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", f.Severity, f.Code, f.Message)
				if f.Severity == schema.SevError {
					failed = true
				}
			}
			if isolation.HasCodeOutside(findings) && !b.Environment.IsolationWarning {
				b.Environment.IsolationWarning = true
				if err := feature.Save(repo, b); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not persist isolation_warning: %v\n", err)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "board: set environment.isolation_warning=true")
				}
			}
			if failed {
				return fmt.Errorf("doctor failed for %q", args[0])
			}
			fmt.Fprintln(cmd.OutOrStdout(), "doctor: warnings only (set isolation.strict=true to fail the gate on these)")
			return nil
		},
	}
	return cmd
}
