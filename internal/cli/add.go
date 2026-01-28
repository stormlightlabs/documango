package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/cache"
	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/ingest/atproto"
	golangingest "github.com/stormlightlabs/documango/internal/ingest/golang"
)

var (
	addVersion  string
	addStart    string
	addMax      int
	addStdlib   bool
	addLexicons bool
)

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <source-type> <source>",
		Short: "Add documentation to the database",
		Long: `Add documentation from various sources to the database.

Supported source types:
  go       - Go module or standard library
  atproto  - AT Protocol specifications and documentation`,
		Example: `  documango add go golang.org/x/net
  documango add go --stdlib
  documango add atproto`,
		Args:              cobra.MinimumNArgs(1),
		RunE:              runAdd,
		ValidArgsFunction: addSourceCompletion,
	}

	cmd.Flags().StringVar(&addVersion, "version", "", "Version for Go modules (module mode) or Go toolchain tag (stdlib mode)")
	cmd.Flags().StringVarP(&addStart, "start", "s", "", "Start at a specific stdlib package path (stdlib mode only)")
	cmd.Flags().IntVarP(&addMax, "max", "m", 0, "Limit number of stdlib packages ingested (stdlib mode only)")
	cmd.Flags().BoolVar(&addStdlib, "stdlib", false, "Use stdlib mode (no module argument)")
	cmd.Flags().BoolVar(&addLexicons, "lexicons-only", false, "Only ingest lexicons (atproto mode only)")

	return cmd
}

func runAdd(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return errors.New("add requires a source type and source identifier")
	}

	sourceType := args[0]
	source := args[1]

	dbPath, err := resolveDBPath()
	if err != nil {
		return err
	}

	if err := db.EnsureDir(dbPath); err != nil {
		return err
	}

	store, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.EnsureSchema(ctx); err != nil {
		return err
	}

	cacheDir, err := cache.CacheDir()
	if err != nil {
		return err
	}
	c, err := cache.New(cacheDir)
	if err != nil {
		if !quiet {
			fmt.Fprintln(os.Stderr, "Warning: cache initialization failed, proceeding without cache:", err)
		}
		c = nil
	}

	switch sourceType {
	case "go":
		return addGoSource(ctx, cmd, store, source, c)
	case "atproto":
		return addAtprotoSource(ctx, cmd, store, c)
	default:
		return fmt.Errorf("unknown source type: %s", sourceType)
	}
}

func addGoSource(ctx context.Context, _ *cobra.Command, store *db.Store, source string, c *cache.FilesystemCache) error {
	if addStdlib {
		if source != "" {
			return errors.New("module argument not allowed with --stdlib")
		}
		opts := golangingest.StdlibOptions{
			DB:          store,
			Version:     addVersion,
			Start:       addStart,
			MaxPackages: addMax,
			Cache:       c,
		}
		if err := golangingest.IngestStdlib(ctx, opts); err != nil {
			return err
		}
		if !quiet {
			p.PrintSuccess("Ingested Go standard library")
		}
		return nil
	}

	if source == "" {
		return errors.New("module argument is required unless --stdlib is set")
	}

	if err := golangingest.IngestModule(ctx, golangingest.Options{
		Module:  source,
		Version: addVersion,
		DB:      store,
		Cache:   c,
	}); err != nil {
		return err
	}

	if !quiet {
		p.PrintSuccess(fmt.Sprintf("Ingested %s", p.FormatSymbol(source)))
	}
	return nil
}

func addAtprotoSource(ctx context.Context, _ *cobra.Command, store *db.Store, c *cache.FilesystemCache) error {
	if err := atproto.IngestAtproto(ctx, atproto.Options{
		DB:    store,
		Cache: c,
	}); err != nil {
		return err
	}

	if !quiet {
		p.PrintSuccess("Ingested AT Protocol documentation")
	}
	return nil
}

func addSourceCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return []string{"go", "atproto"}, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}
