package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/stormlightlabs/documango/internal/db"
)

// Run starts the Bubble Tea program with the given database store.
func Run(store *db.Store) error {
	p := tea.NewProgram(NewRootModel(store), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
