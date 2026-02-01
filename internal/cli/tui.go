package cli

import (
	"github.com/spf13/cobra"

	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/tui"
)

func newTuiCommand() *cobra.Command {
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the terminal user interface",
		Long:  `Launch the interactive terminal-based documentation browser.`,
		RunE:  runTui,
	}
	return tuiCmd
}

func runTui(cmd *cobra.Command, args []string) error {
	dbPath, err := resolveDBPath()
	if err != nil {
		return err
	}

	store, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	return tui.Run(store)
}
