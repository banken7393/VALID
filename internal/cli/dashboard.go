package cli

import (
	"fmt"

	"github.com/banken7393/valid/internal/dashboard"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newDashboardCmd() *cobra.Command {
	var (
		port    int
		feature string
	)
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Serve the visual progress dashboard (loopback only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			cfg, err := schema.LoadConfig(repo)
			if err != nil {
				return err
			}
			if port <= 0 {
				port = cfg.DashboardPort
			}
			if feature == "" {
				slugs, err := schema.ListFeatureSlugs(repo)
				if err != nil {
					return err
				}
				if len(slugs) == 0 {
					return fmt.Errorf("no feature boards found; pass --feature <slug>")
				}
				if len(slugs) > 1 {
					return fmt.Errorf("multiple feature boards %v; pass --feature <slug>", slugs)
				}
				feature = slugs[0]
			}
			dataPath := schema.BoardPath(repo, feature)
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			srv := dashboard.NewWithRepo(dataPath, repo, addr)
			return srv.ListenAndServe()
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "HTTP port (default from config)")
	cmd.Flags().StringVar(&feature, "feature", "", "Feature slug to display")
	return cmd
}
