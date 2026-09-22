package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/banken7393/valid/internal/feature"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newBoardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "board",
		Short: "Optional helper: persist board fields (prefer LLM file edits)",
	}
	cmd.AddCommand(newBoardSetCmd())
	cmd.AddCommand(newBoardShowCmd())
	return cmd
}

func newBoardShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <slug>",
		Short: "Print board JSON",
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
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(b)
		},
	}
}

func newBoardSetCmd() *cobra.Command {
	var (
		lifecycle  string
		autonomy   string
		north      string
		howFile    string
		howInline  string
		whatJSON   string
		phasesJSON string
		decision   string
	)

	cmd := &cobra.Command{
		Use:   "set <slug>",
		Short: "Update board fields atomically",
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
			if north != "" {
				b.NorthStar = north
			}
			if lifecycle != "" {
				if err := b.SetLifecycle(lifecycle); err != nil {
					return err
				}
			}
			if autonomy != "" {
				if err := b.SetAutonomy(autonomy); err != nil {
					return err
				}
			}
			if howInline != "" {
				b.HowItWorks = howInline
			}
			if howFile != "" {
				raw, err := os.ReadFile(howFile)
				if err != nil {
					return err
				}
				b.HowItWorks = string(raw)
				dest := schema.FeatureDir(repo, slug) + "/how-it-works.mmd"
				if err := os.WriteFile(dest, raw, 0o644); err != nil {
					return err
				}
			}
			if whatJSON != "" {
				var items []schema.AcceptanceCriterion
				if err := json.Unmarshal([]byte(whatJSON), &items); err != nil {
					return fmt.Errorf("parse --what JSON: %w", err)
				}
				if err := b.SetWhat(items); err != nil {
					return err
				}
			}
			if phasesJSON != "" {
				var phases []schema.Phase
				if err := json.Unmarshal([]byte(phasesJSON), &phases); err != nil {
					return fmt.Errorf("parse --phases JSON: %w", err)
				}
				b.Phases = phases
			}
			if decision != "" {
				// format id|title|detail
				parts := strings.SplitN(decision, "|", 3)
				id := parts[0]
				title, detail := id, ""
				if len(parts) > 1 {
					title = parts[1]
				}
				if len(parts) > 2 {
					detail = parts[2]
				}
				if err := b.AddDecision(id, title, detail, nil); err != nil {
					return err
				}
			}
			b.Touch()
			if err := feature.Save(repo, b); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated board %q lifecycle=%s autonomy=%s\n", slug, b.Lifecycle, b.Autonomy)
			return nil
		},
	}

	cmd.Flags().StringVar(&lifecycle, "lifecycle", "", "plan|spec|build|audit|finish|done")
	cmd.Flags().StringVar(&autonomy, "autonomy", "", "in_the_loop|above_the_loop")
	cmd.Flags().StringVar(&north, "north-star", "", "North star text")
	cmd.Flags().StringVar(&howFile, "how-it-works-file", "", "Path to Mermaid file")
	cmd.Flags().StringVar(&howInline, "how-it-works", "", "Inline Mermaid source")
	cmd.Flags().StringVar(&whatJSON, "what", "", `JSON array of {"id","description"} ACs`)
	cmd.Flags().StringVar(&phasesJSON, "phases", "", `JSON array of {"id","name","outcome","status"}`)
	cmd.Flags().StringVar(&decision, "decision", "", "id|title|detail")
	return cmd
}
