package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/config"
)

var (
	cfg     *config.Config
	dbPath  string
	verbose bool
	quiet   bool
	noColor bool
)

var rootCmd = &cobra.Command{
	Use:   "documango",
	Short: "A terminal-first documentation browser",
	Long: `Documango is a terminal-first documentation browser that ingests,
stores, and searches technical documentation from various sources including
Go modules, Go stdlib, and AT Protocol specifications.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	loadConfig()
	rootCmd.AddCommand(
		newInitCommand(),
		newAddCommand(),
		newSearchCommand(),
		newReadCommand(),
		newListCommand(),
		newInfoCommand(),
		newCacheCommand(),
		newConfigCommand(),
	)
	return rootCmd.Execute()
}

func loadConfig() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dbPath, "database", "d", "", "Database path (default: $XDG_DATA_HOME/documango/default.usde)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	if noColor {
		rootCmd.CompletionOptions.DisableDescriptions = true
	}
}

func resolveDBPath() (string, error) {
	if dbPath != "" {
		return config.ResolveDatabasePath(dbPath)
	}
	return config.GetDefaultDatabase()
}
