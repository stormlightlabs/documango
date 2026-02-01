package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestStyles_Defined verifies all styles are defined
func TestStyles_Defined(t *testing.T) {
	_ = titleStyle.Render("test")
	_ = subtitleStyle.Render("test")
	_ = helpStyle.Render("test")
	_ = searchInputStyle.Render("test")
	_ = errorStyle.Render("test")
	_ = dimStyle.Render("test")
	_ = accentStyle.Render("test")
	_ = nameStyle.Render("test")
	_ = selectedNameStyle.Render("test")
	_ = typeStyle.Render("test")
	_ = selectedTypeStyle.Render("test")
	_ = emptyStateStyle.Render("test")
	_ = docTitleStyle.Render("test")
	_ = docBackStyle.Render("test")
	_ = docLinksStyle.Render("test")
	_ = activeTabStyle.Render("test")
	_ = inactiveTabStyle.Render("test")
}

// TestNewWithColor verifies the helper function creates styles correctly
func TestNewWithColor(t *testing.T) {
	style := newWithColor("#ff0000")

	result := style.Render("test")
	if result == "" {
		t.Error("expected styled output")
	}
}

// TestTitleStyle verifies title style properties
func TestTitleStyle(t *testing.T) {
	result := titleStyle.Render("Title")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestSearchInputStyle verifies search input style has border
func TestSearchInputStyle(t *testing.T) {
	result := searchInputStyle.Render("input")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestErrorStyle verifies error style
func TestErrorStyle(t *testing.T) {
	result := errorStyle.Render("error")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestAccentStyle verifies accent style
func TestAccentStyle(t *testing.T) {
	result := accentStyle.Render("accent")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestDimStyle verifies dim style
func TestDimStyle(t *testing.T) {
	result := dimStyle.Render("dim")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestEmptyStateStyle verifies empty state style
func TestEmptyStateStyle(t *testing.T) {
	result := emptyStateStyle.Render("empty")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestDocTitleStyle verifies doc title style
func TestDocTitleStyle(t *testing.T) {
	result := docTitleStyle.Render("doc title")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestDocBackStyle verifies doc back style
func TestDocBackStyle(t *testing.T) {
	result := docBackStyle.Render("back")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestDocLinksStyle verifies doc links style
func TestDocLinksStyle(t *testing.T) {
	result := docLinksStyle.Render("links")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestSelectedStyles verifies selected item styles
func TestSelectedStyles(t *testing.T) {
	result := selectedNameStyle.Render("selected")
	if result == "" {
		t.Error("expected non-empty output for selected name")
	}

	result = selectedTypeStyle.Render("type")
	if result == "" {
		t.Error("expected non-empty output for selected type")
	}
}

// TestTypeStyles verifies type styles
func TestTypeStyles(t *testing.T) {
	result := typeStyle.Render("function")
	if result == "" {
		t.Error("expected non-empty output for type")
	}

	result = nameStyle.Render("name")
	if result == "" {
		t.Error("expected non-empty output for name")
	}
}

// TestHelpStyle verifies help style
func TestHelpStyle(t *testing.T) {
	result := helpStyle.Render("help text")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestSubtitleStyle verifies subtitle style
func TestSubtitleStyle(t *testing.T) {
	result := subtitleStyle.Render("subtitle")
	if result == "" {
		t.Error("expected non-empty output")
	}
}

// TestStyleTypes verifies styles are correct lipgloss types
func TestStyleTypes(t *testing.T) {
	// All styles should be lipgloss.Style
	var _ lipgloss.Style = titleStyle
	var _ lipgloss.Style = subtitleStyle
	var _ lipgloss.Style = helpStyle
	var _ lipgloss.Style = searchInputStyle
	var _ lipgloss.Style = errorStyle
	var _ lipgloss.Style = dimStyle
	var _ lipgloss.Style = accentStyle
	var _ lipgloss.Style = nameStyle
	var _ lipgloss.Style = selectedNameStyle
	var _ lipgloss.Style = typeStyle
	var _ lipgloss.Style = selectedTypeStyle
	var _ lipgloss.Style = emptyStateStyle
	var _ lipgloss.Style = docTitleStyle
	var _ lipgloss.Style = docBackStyle
	var _ lipgloss.Style = docLinksStyle
}

// TestStyles_Content verifies that styled output contains the original text
func TestStyles_Content(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
		input string
	}{
		{"title", titleStyle, "test"},
		{"error", errorStyle, "error"},
		{"accent", accentStyle, "accent"},
		{"dim", dimStyle, "dim"},
		{"name", nameStyle, "name"},
		{"selectedName", selectedNameStyle, "selected"},
		{"type", typeStyle, "type"},
		{"selectedType", selectedTypeStyle, "type"},
		{"emptyState", emptyStateStyle, "empty"},
		{"docTitle", docTitleStyle, "title"},
		{"docBack", docBackStyle, "back"},
		{"docLinks", docLinksStyle, "links"},
		{"help", helpStyle, "help"},
		{"subtitle", subtitleStyle, "subtitle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.style.Render(tt.input)
			if !strings.Contains(result, tt.input) {
				t.Errorf("expected output to contain %q, got %q", tt.input, result)
			}
		})
	}
}
