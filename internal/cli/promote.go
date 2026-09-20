package cli

import (
	"fmt"

	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/promote"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newPromoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote knowledge or env deltas (plumbing; skills call this)",
	}
	cmd.AddCommand(newPromoteKnowledgeCmd())
	cmd.AddCommand(newPromoteEnvCmd())
	return cmd
}

func newPromoteKnowledgeCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "knowledge <slug>",
		Short: "Merge board knowledge into MCP corpus + write short ficha",
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
			res, err := promote.PromoteKnowledge(repo, b, cfg, dryRun)
			if err != nil {
				return err
			}
			if !dryRun {
				if err := feature.Save(repo, b); err != nil {
					return err
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), res.Message)
			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "dry-run docs: %v\n", append(res.UpdatedDocs, res.CreatedDocs...))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show actions without writing")
	return cmd
}

func newPromoteEnvCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "env <slug>",
		Short: "Apply pending env/deps to main Devcontainer (fail hard on conflict)",
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
			res, err := promote.PromoteEnv(repo, b, cfg, dryRun)
			if res != nil && res.Diff != "" {
				fmt.Fprintln(cmd.OutOrStdout(), res.Diff)
			}
			if err != nil {
				return err
			}
			if !dryRun {
				if err := feature.Save(repo, b); err != nil {
					return err
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), res.Message)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show diff without applying")
	return cmd
}
