package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/banken7393/valid/internal/mcp"
	"github.com/banken7393/valid/internal/rag"
	"github.com/banken7393/valid/internal/schema"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	var (
		httpMode      bool
		addr          string
		knowledge     string
		token         string
		allowScripts  bool
	)

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the VALID MCP server (knowledge RAG + project scripts)",
		Long: `Exposes:
  - RAG: search_knowledge, list_documents, get_document, upsert_document
  - Project scripts (stdio by default): one MCP tool per .valid/scripts/<id>/meta.json

HTTP mode is loopback-only, requires a bearer token (VALID_MCP_TOKEN or --token),
and disables project scripts unless --allow-scripts-http is set.

Stdio (default): valid mcp
HTTP: VALID_MCP_TOKEN=… valid mcp --http --addr 127.0.0.1:7433`,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := resolveRepo()
			if err != nil {
				return err
			}
			cfg, err := schema.LoadConfig(repo)
			if err != nil {
				return err
			}
			root := knowledge
			if root == "" {
				root = schema.KnowledgePath(repo, cfg)
			}
			scriptsRoot := schema.ScriptsPath(repo, cfg)

			idx := rag.NewIndex(root)
			if err := idx.Load(); err != nil {
				return err
			}

			tok := strings.TrimSpace(token)
			if tok == "" {
				tok = strings.TrimSpace(os.Getenv("VALID_MCP_TOKEN"))
			}

			opts := mcp.Options{
				RepoRoot:    repo,
				ScriptsRoot: scriptsRoot,
			}
			if httpMode {
				opts.ScriptsDisabled = !allowScripts
				opts.HTTPToken = tok
			}

			srv := mcp.NewServer(idx, opts)
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			listen := addr
			if httpMode && !cmd.Flags().Changed("addr") {
				listen = cfg.ResolveMCPHTTPAddr()
			}
			if httpMode {
				fmt.Fprintf(os.Stderr, "VALID MCP HTTP on http://%s/mcp (knowledge: %s scripts=%v)\n", strings.TrimPrefix(listen, ""), root, allowScripts)
				return srv.ServeHTTP(ctx, listen)
			}
			fmt.Fprintf(os.Stderr, "VALID MCP stdio (knowledge: %s scripts: %s)\n", root, scriptsRoot)
			return srv.ServeStdio(ctx)
		},
	}

	cmd.Flags().BoolVar(&httpMode, "http", false, "Serve MCP over HTTP/SSE instead of stdio")
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:7433", "HTTP listen address (loopback only)")
	cmd.Flags().StringVar(&knowledge, "knowledge", "", "Override path to knowledge root")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token for HTTP (or VALID_MCP_TOKEN)")
	cmd.Flags().BoolVar(&allowScripts, "allow-scripts-http", false, "Allow project script tools over HTTP (dangerous; prefer stdio)")
	return cmd
}
