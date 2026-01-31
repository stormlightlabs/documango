package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/config"
)

var (
	cfg     *config.Config
	dbPath  string
	verbose bool
	quiet   bool
	noColor bool
	p       *Printer = NewPrinter()
)

var rootCmd = &cobra.Command{
	Use:   "documango",
	Short: "A terminal-first documentation browser",
	Long: `Documango is a terminal-first documentation browser that ingests,
stores, and searches technical documentation from various sources.

It supports:

- Rust crates (docs.rs)
- Go modules (go.dev/doc) & the standard library
- AT Protocol specifications
- Hex.pm docs for Elixir & Gleam packages
- Github repository markdown files
`,
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
		newMCPCommand(),
		newWebCommand(),
	)
	return rootCmd.Execute()
}

func loadConfig() {
	var err error
	if cfg, err = config.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}
}

func resolveDBPath() (string, error) {
	if dbPath != "" {
		return config.ResolveDatabasePath(dbPath)
	}
	return config.GetDefaultDatabase()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dbPath, "database", "d", "", "Database path (default: $XDG_DATA_HOME/documango/default.usde)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	if noColor {
		rootCmd.CompletionOptions.DisableDescriptions = true
	}
	cobra.OnInitialize(func() {
		if noColor {
			lipgloss.SetColorProfile(termenv.Ascii)
		}
	})
}
