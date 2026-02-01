package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle        = newWithColor("#22c55e").Bold(true).MarginBottom(1)
	subtitleStyle     = newWithColor("#a1a1a1").MarginBottom(2)
	helpStyle         = newWithColor("#525252").Italic(true)
	searchInputStyle  = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#404040"))
	errorStyle        = newWithColor("#ef4444")
	dimStyle          = newWithColor("#525252").Italic(true)
	accentStyle       = newWithColor("#22c55e")
	nameStyle         = newWithColor("#fafafa")
	selectedNameStyle = newWithColor("#22c55e").Bold(true)
	typeStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#737373")).Faint(true)
	selectedTypeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	emptyStateStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#737373")).Italic(true).Padding(1, 2)
	docTitleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Bold(true)
	docBackStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#737373"))
	docLinksStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#737373")).Italic(true)
)

func newWithColor(c string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
}
