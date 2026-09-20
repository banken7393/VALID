package cli

import (
	"fmt"

	"github.com/banken7393/valid/internal/templates"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize .valid/ (config, knowledge, skills, agents, scripts)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			if err := templates.InitProject(repo, templates.InitOptions{Force: force}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized VALID in %s/.valid\n", repo)
			fmt.Fprintln(cmd.OutOrStdout(), "  config.json, knowledge/, skills/, agents/, scripts/, features/")
			fmt.Fprintln(cmd.OutOrStdout(), "  Method UX: skills plan → spec → build → audit → finish (+ patch/minipatch/delegate)")
			fmt.Fprintln(cmd.OutOrStdout(), "  Agents: interviewer, developer, reviewer, delegate · autonomy in_the_loop|above_the_loop")
			fmt.Fprintln(cmd.OutOrStdout(), "  MCP scripts: add folders under .valid/scripts/<id>/ with meta.json")
			fmt.Fprintln(cmd.OutOrStdout(), "  Wire your IDE (optional): valid install cursor   # or claude|opencode|codex|all")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing skill/agent/knowledge/script seed files")
	return cmd
}
