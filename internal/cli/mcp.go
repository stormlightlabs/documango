package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/mcp"
)

func newMCPCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "mcp", Short: "Model Context Protocol server"}
	cmd.AddCommand(newMCPServeCommand())
	return cmd
}

func newMCPServeCommand() *cobra.Command {
	var stdio bool
	var httpAddr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !stdio && httpAddr == "" {
				stdio = true
			}

			path, err := resolveDBPath()
			if err != nil {
				return err
			}

			store, err := db.Open(path)
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer store.Close()

			server := mcp.NewServer(store, "0.1.0")
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			if httpAddr != "" {
				fmt.Fprintf(os.Stderr, "Starting MCP server on HTTP %s\n", httpAddr)
				return mcp.RunHTTP(ctx, server, httpAddr)
			}
			return mcp.RunStdio(ctx, server)
		},
	}

	cmd.Flags().BoolVar(&stdio, "stdio", false, "Use stdio transport (default)")
	cmd.Flags().StringVar(&httpAddr, "http", "", "Use HTTP transport on the specified address (e.g., :8080)")
	return cmd
}
