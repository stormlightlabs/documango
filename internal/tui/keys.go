package tui

import "github.com/charmbracelet/bubbles/key"

// keyBindings defines application-wide key bindings.
type keyBindings struct {
	Quit     key.Binding
	Search   key.Binding
	Navigate key.Binding
	Open     key.Binding
	Back     key.Binding
	Scroll   key.Binding
	Help     key.Binding
	Link     key.Binding
}

// newKeyBindings creates a new key binding set.
func newKeyBindings() keyBindings {
	return keyBindings{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Navigate: key.NewBinding(
			key.WithKeys("j", "k", "up", "down", "g", "G"),
			key.WithHelp("j/k/g/G", "navigate"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Scroll: key.NewBinding(
			key.WithKeys("j", "k", "d", "u", "g", "G"),
			key.WithHelp("j/k/d/u/g/G", "scroll"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Link: key.NewBinding(
			key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
			key.WithHelp("1-9", "link"),
		),
	}
}

// ShortHelp returns key bindings for the short help overlay.
func (k keyBindings) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Search,
		k.Navigate,
		k.Open,
		k.Back,
		k.Help,
		k.Quit,
	}
}

// FullHelp returns key bindings for the full help overlay.
func (k keyBindings) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Search, k.Navigate, k.Open, k.Back},
		{k.Scroll, k.Link, k.Help, k.Quit},
	}
}
