package cli

import (
	"fmt"

	"github.com/banken7393/valid/internal/ide"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var force bool
	var link bool
	cmd := &cobra.Command{
		Use:   "install [cursor|claude|opencode|codex|all]",
		Short: "Wire VALID skills, agents, and MCP into your IDE",
		Long: `Install IDE adapters for VALID.

Copies (or --link symlinks) method skills into the IDE skill tree and writes
MCP / rules / AGENTS fragments so you can invoke /plan, /spec, … and use
valid mcp from the editor.

Targets:
  cursor    .cursor/mcp.json, rules, skills/, agents/
  claude    .mcp.json, .claude/CLAUDE.md, skills/, agents/
  opencode  .opencode.valid.json + skills/agents (merge fragment into OpenCode)
  codex     .codex/valid.mcp.toml, AGENTS.md, skills/, agents/
  all       every target above (default)

Skills are always refreshed from .valid/skills/. Adapter files (mcp.json, …)
are kept unless --force.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			target := "all"
			if len(args) == 1 {
				target = args[0]
			}
			reports, err := ide.Install(repo, target, ide.Options{Force: force, Link: link})
			if err != nil {
				return err
			}
			for _, r := range reports {
				fmt.Fprintf(cmd.OutOrStdout(), "[%s]\n", r.Target)
				for _, a := range r.Actions {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", a)
				}
			}
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), "Next: invoke skills manually in chat (/plan, /spec, /build, /audit, /finish).")
			fmt.Fprintln(cmd.OutOrStdout(), "Skills are disable-model-invocation — the model will not auto-pick them.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing MCP/rules/AGENTS adapter files")
	cmd.Flags().BoolVar(&link, "link", false, "Symlink skills/agents instead of copying")
	return cmd
}
