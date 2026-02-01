package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stormlightlabs/documango/internal/db"
)

// TestListModel_Init verifies list model initializes correctly
func TestListModel_Init(t *testing.T) {
	model := NewListModel()
	if cmd := model.Init(); cmd != nil {
		t.Error("expected Init to return nil command")
	}
}

// TestListModel_View_Empty tests the view with no items
func TestListModel_View_Empty(t *testing.T) {
	model := NewListModel()
	view := model.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	if !strings.Contains(view, "No results found") {
		t.Error("expected view to show empty state message")
	}
}

// TestListModel_View_WithItems tests the view with items
func TestListModel_View_WithItems(t *testing.T) {
	model := NewListModel()
	results := []db.SearchResult{
		{Name: "Item 1", Type: "function", DocID: 1},
		{Name: "Item 2", Type: "struct", DocID: 2},
	}
	model.SetResults(results)

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view with items")
	}

	if strings.Contains(view, "No results found") {
		t.Error("expected view to not show empty state when items exist")
	}
}

// TestListModel_Update_Navigation tests navigation keys
func TestListModel_Update_Navigation(t *testing.T) {
	model := NewListModel()
	results := []db.SearchResult{
		{Name: "Item 1", Type: "function", DocID: 1},
		{Name: "Item 2", Type: "struct", DocID: 2},
		{Name: "Item 3", Type: "method", DocID: 3},
	}
	model.SetResults(results)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, cmd := model.Update(msg)
	if cmd != nil {
		t.Error("expected no command from navigation")
	}
	_ = newModel

	msg = tea.KeyMsg{Type: tea.KeyDown}
	newModel, cmd = model.Update(msg)
	if cmd != nil {
		t.Error("expected no command from navigation")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from navigation")
	}

	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from navigation")
	}
}

// TestListModel_Update_GotoStartEnd tests goto start/end keys
func TestListModel_Update_GotoStartEnd(t *testing.T) {
	model := NewListModel()
	results := []db.SearchResult{
		{Name: "Item 1", Type: "function", DocID: 1},
		{Name: "Item 2", Type: "struct", DocID: 2},
		{Name: "Item 3", Type: "method", DocID: 3},
	}
	model.SetResults(results)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	newModel, cmd := model.Update(msg)
	if cmd != nil {
		t.Error("expected no command from G key")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from g key")
	}
	_ = newModel
}

// TestListModel_Update_Select tests selecting an item
func TestListModel_Update_Select(t *testing.T) {
	model := NewListModel()
	results := []db.SearchResult{
		{Name: "Item 1", Type: "function", DocID: 1},
	}
	model.SetResults(results)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected command from enter key (select)")
	}

	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(listSelectMsg); !ok {
			t.Error("expected listSelectMsg from enter key")
		}
	}

	_ = newModel
}

// TestListModel_Update_FocusSearch tests the focus search key
func TestListModel_Update_FocusSearch(t *testing.T) {
	model := NewListModel()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newModel, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected command from / key")
	}

	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(focusSearchMsg); !ok {
			t.Error("expected focusSearchMsg from / key")
		}
	}

	_ = newModel
}

// TestListModel_Update_SearchResults tests handling search results
func TestListModel_Update_SearchResults(t *testing.T) {
	model := NewListModel()
	results := []db.SearchResult{
		{Name: "Result 1", Type: "function", DocID: 1},
		{Name: "Result 2", Type: "struct", DocID: 2},
	}
	msg := searchResultsMsg{results: results, query: "test"}
	newModel, cmd := model.Update(msg)
	if cmd != nil {
		t.Error("expected no command from search results")
	}

	m := newModel
	if len(m.results) != 2 {
		t.Errorf("expected 2 results, got %d", len(m.results))
	}
}

// TestListModel_SetResults tests setting results
func TestListModel_SetResults(t *testing.T) {
	model := NewListModel()

	results := []db.SearchResult{
		{Name: "Result 1", Type: "function", DocID: 1},
		{Name: "Result 2", Type: "struct", DocID: 2},
	}
	model.SetResults(results)

	if len(model.results) != 2 {
		t.Errorf("expected 2 results, got %d", len(model.results))
	}

	if len(model.list.Items()) != 2 {
		t.Errorf("expected 2 list items, got %d", len(model.list.Items()))
	}
}

// TestListModel_Selected tests getting selected item
func TestListModel_Selected(t *testing.T) {
	model := NewListModel()

	if model.Selected() != nil {
		t.Error("expected no selection initially")
	}

	results := []db.SearchResult{
		{Name: "Result 1", Type: "function", DocID: 1},
	}
	model.SetResults(results)
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(msg)
	model = newModel
}

// TestListModel_ClearSelection tests clearing selection
func TestListModel_ClearSelection(t *testing.T) {
	model := NewListModel()
	model.selected = &db.SearchResult{Name: "Test", Type: "function", DocID: 1}
	model.ClearSelection()
	if model.Selected() != nil {
		t.Error("expected selection to be cleared")
	}
}

// TestResultItem_FilterValue tests filter value implementation
func TestResultItem_FilterValue(t *testing.T) {
	result := db.SearchResult{Name: "TestName", Type: "function", DocID: 1}
	item := NewResultItem(result)

	if item.FilterValue() != "TestName" {
		t.Errorf("expected FilterValue 'TestName', got %q", item.FilterValue())
	}
}

// TestResultItem_Result tests getting the underlying result
func TestResultItem_Result(t *testing.T) {
	result := db.SearchResult{Name: "TestName", Type: "function", DocID: 1}
	item := NewResultItem(result)
	r := item.Result()
	if r.Name != "TestName" {
		t.Errorf("expected Name 'TestName', got %q", r.Name)
	}
	if r.Type != "function" {
		t.Errorf("expected Type 'function', got %q", r.Type)
	}
	if r.DocID != 1 {
		t.Errorf("expected DocID 1, got %d", r.DocID)
	}
}

// TestResultDelegate tests the result delegate
func TestResultDelegate(t *testing.T) {
	delegate := NewResultDelegate()

	if delegate.Height() != 2 {
		t.Errorf("expected height 2, got %d", delegate.Height())
	}

	if delegate.Spacing() != 1 {
		t.Errorf("expected spacing 1, got %d", delegate.Spacing())
	}

	cmd := delegate.Update(nil, nil)
	if cmd != nil {
		t.Error("expected Update to return nil")
	}
}
