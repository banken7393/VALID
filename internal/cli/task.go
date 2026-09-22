package cli

import (
	"fmt"
	"strings"

	"github.com/banken7393/valid/internal/feature"
	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	var (
		title  string
		status string
		covers string
		desc   string
		phase  string
	)
	cmd := &cobra.Command{
		Use:   "task <slug> <task-id>",
		Short: "Optional helper: upsert a task (prefer LLM file edits)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			slug, taskID := args[0], args[1]
			b, err := feature.Load(repo, slug)
			if err != nil {
				return err
			}
			var coverIDs []string
			if covers != "" {
				for _, c := range strings.Split(covers, ",") {
					c = strings.TrimSpace(c)
					if c != "" {
						coverIDs = append(coverIDs, c)
					}
				}
			}
			if err := b.UpsertTaskFull(taskID, title, status, desc, phase, coverIDs); err != nil {
				return err
			}
			if b.Lifecycle == "plan" || b.Lifecycle == "spec" {
				_ = b.SetLifecycle("build")
			}
			if err := feature.Save(repo, b); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Upserted task %q on %q phase=%q covers=%v\n", taskID, slug, phase, coverIDs)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Task title")
	cmd.Flags().StringVar(&status, "status", "pending", "pending|doing|done|failed")
	cmd.Flags().StringVar(&covers, "covers", "", "Comma-separated AC ids")
	cmd.Flags().StringVar(&desc, "description", "", "Optional description")
	cmd.Flags().StringVar(&phase, "phase", "", "Phase id (phases[].id), e.g. p1")
	return cmd
}
