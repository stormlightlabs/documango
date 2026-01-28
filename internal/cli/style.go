package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

type palette struct {
	Base00 lipgloss.Color // Background
	Base01 lipgloss.Color // Surface
	Base02 lipgloss.Color // Selection
	Base03 lipgloss.Color // Muted
	Base04 lipgloss.Color // Subtle
	Base05 lipgloss.Color // Text
	Base06 lipgloss.Color // Bright Text
	Base07 lipgloss.Color
	Base08 lipgloss.Color
	Base09 lipgloss.Color
	Base0A lipgloss.Color
	Base0B lipgloss.Color
	Base0C lipgloss.Color
	Base0D lipgloss.Color
	Base0E lipgloss.Color
	Base0F lipgloss.Color
}

var (
	cyan   = lipgloss.Color("#08bdba")
	teal   = lipgloss.Color("#3ddbd9")
	blue1  = lipgloss.Color("#78a9ff")
	pink   = lipgloss.Color("#ee5396")
	green  = lipgloss.Color("#42be65")
	purple = lipgloss.Color("#be95ff")
	blue2  = lipgloss.Color("#33b1ff")
	pink2  = lipgloss.Color("#ff7eb6")
	blue3  = lipgloss.Color("#82cfff")

	// Oxocarbon Dark Palette
	//
	// Source: https://github.com/nyoom-engineering/oxoc
	oxoc = palette{
		Base00: lipgloss.Color("#161616"),
		Base01: lipgloss.Color("#262626"),
		Base02: lipgloss.Color("#393939"),
		Base03: lipgloss.Color("#525252"),
		Base04: lipgloss.Color("#dde1e6"),
		Base05: lipgloss.Color("#f2f4f8"),
		Base06: lipgloss.Color("#ffffff"),
		Base07: cyan,
		Base08: teal,
		Base09: blue1,
		Base0A: pink,
		Base0B: blue2,
		Base0C: pink2,
		Base0D: green,
		Base0E: purple,
		Base0F: blue3,
	}
)

// Styles wraps the lipgloss styles for the application.
type Styles struct {
	Header  lipgloss.Style
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Muted   lipgloss.Style
	Accent  lipgloss.Style
	Link    lipgloss.Style
	Path    lipgloss.Style
	Code    lipgloss.Style
}

// NewStyles returns a new Styles struct with Oxocarbon defaults.
func NewStyles() *Styles {
	return &Styles{
		Header:  lipgloss.NewStyle().Foreground(oxoc.Base0E).Bold(true),
		Success: lipgloss.NewStyle().Foreground(oxoc.Base0D),
		Error:   lipgloss.NewStyle().Foreground(oxoc.Base0C),
		Warning: lipgloss.NewStyle().Foreground(oxoc.Base0A),
		Info:    lipgloss.NewStyle().Foreground(oxoc.Base09),
		Muted:   lipgloss.NewStyle().Foreground(oxoc.Base03),
		Accent:  lipgloss.NewStyle().Foreground(oxoc.Base07),
		Link:    lipgloss.NewStyle().Foreground(oxoc.Base0B).Underline(true),
		Path:    lipgloss.NewStyle().Foreground(oxoc.Base08),
		Code:    lipgloss.NewStyle().Foreground(oxoc.Base05).Background(oxoc.Base01).Padding(0, 1),
	}
}

// Printer provides helper methods for printing formatted output.
type Printer struct {
	Styles *Styles
}

// NewPrinter creates a new Printer with default Oxocarbon styles.
func NewPrinter() *Printer {
	return &Printer{Styles: NewStyles()}
}

// PrintHeader prints a bold header message.
func (p *Printer) PrintHeader(msg string) {
	fmt.Println(p.Styles.Header.Render(msg))
}

// PrintSuccess prints a success message with a checkmark.
func (p *Printer) PrintSuccess(msg string) {
	fmt.Printf("%s %s\n", p.Styles.Success.Render("✔"), msg)
}

// PrintError prints an error message to stderr with a cross.
func (p *Printer) PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", p.Styles.Error.Render("✘"), msg)
}

// PrintWarning prints a warning message with an exclamation.
func (p *Printer) PrintWarning(msg string) {
	fmt.Printf("%s %s\n", p.Styles.Warning.Render("⚠"), msg)
}

// PrintInfo prints an info message with an 'i' symbol.
func (p *Printer) PrintInfo(msg string) {
	fmt.Printf("%s %s\n", p.Styles.Info.Render("ℹ"), msg)
}

// PrintListItem prints a muted label with a value.
func (p *Printer) PrintListItem(label, value string) {
	fmt.Printf("%s: %s\n", p.Styles.Muted.Render(label), value)
}

// FormatPath formats a file or document path.
func (p *Printer) FormatPath(path string) string {
	return p.Styles.Path.Render(path)
}

// FormatSymbol formats a code symbol.
func (p *Printer) FormatSymbol(sym string) string {
	return p.Styles.Accent.Render(sym)
}
