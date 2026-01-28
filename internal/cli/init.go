package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/config"
	"github.com/stormlightlabs/documango/internal/db"
)

var initPath string

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [database-name]",
		Short: "Initialize a new documentation database",
		Long: `Initialize a new documentation database (.usde file).

If no database name is provided, creates the default database.
The database will be created in the XDG data directory unless an
absolute path is provided with --path.`,
		Example: `  documango init
  documango init mydocs
  documango init -p /tmp/docs.usde`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInit,
	}

	cmd.Flags().StringVarP(&initPath, "path", "p", "", "Explicit path for database file")
	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	var targetPath string
	var dbName string
	var err error

	if dbPath != "" {
		targetPath = dbPath
		dbName = filepath.Base(dbPath)
	} else if initPath != "" {
		targetPath = initPath
		dbName = filepath.Base(initPath)
	} else if len(args) > 0 {
		dbName = args[0]
		targetPath, err = config.ResolveDatabasePath(args[0])
		if err != nil {
			return err
		}
	} else {
		dbName = "default"
		targetPath, err = config.GetDefaultDatabase()
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("database already exists: %s", targetPath)
	}

	if err := config.EnsureDatabaseDir(targetPath); err != nil {
		return err
	}

	store, err := db.Open(targetPath)
	if err != nil {
		return err
	}
	defer store.Close()

	ctx := context.Background()
	if err := store.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	registry, err := config.LoadRegistry()
	if err == nil {
		_ = registry.Add(dbName, targetPath)
		_ = registry.Save()
	}

	if !quiet {
		p.PrintSuccess(fmt.Sprintf("Initialized %s", p.FormatPath(targetPath)))
	}

	return nil
}
