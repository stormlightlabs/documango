package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/web"
)

var (
	webAddr string
)

func newWebCommand() *cobra.Command {
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Web interface commands",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web documentation server",
		RunE:  runWebServe,
	}

	serveCmd.Flags().StringVar(&webAddr, "http", ":8080", "HTTP service address")

	webCmd.AddCommand(serveCmd)
	return webCmd
}

func runWebServe(cmd *cobra.Command, args []string) error {
	dbPath, err := resolveDBPath()
	if err != nil {
		return err
	}

	store, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	srv := web.NewServer(store, webAddr)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return srv.Start(ctx)
}
