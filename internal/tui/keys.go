package tui

import "github.com/charmbracelet/bubbles/key"

// keyBindings defines application-wide key bindings.
type keyBindings struct {
	Quit key.Binding
}

// newKeyBindings creates a new key binding set.
func newKeyBindings() keyBindings {
	return keyBindings{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns key bindings for the short help overlay.
func (k keyBindings) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Quit,
	}
}

// FullHelp returns key bindings for the full help overlay.
func (k keyBindings) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit},
	}
}
