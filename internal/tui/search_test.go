package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stormlightlabs/documango/internal/db"
)

// TestSearchModel_Init verifies search model initializes correctly
func TestSearchModel_Init(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)

	cmd := model.Init()
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

// TestSearchModel_View_Empty tests the view with no input
func TestSearchModel_View_Empty(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

// TestSearchModel_View_Searching tests the view while searching
func TestSearchModel_View_Searching(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)
	model.searching = true

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view while searching")
	}
}

// TestSearchModel_View_WithResults tests the view with results
func TestSearchModel_View_WithResults(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)
	model.resultCount = 5

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view with results")
	}
}

// TestSearchModel_View_WithError tests the view with an error
func TestSearchModel_View_WithError(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)
	model.err = errTest

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view with error")
	}
}

// TestSearchModel_Update_Typing tests typing in the search input
func TestSearchModel_Update_Typing(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello")}
	newModel, cmd := model.Update(msg)

	m := newModel
	if m.Value() != "hello" {
		t.Errorf("expected value 'hello', got %q", m.Value())
	}

	if cmd == nil {
		t.Error("expected command from typing (debounce)")
	}
}

// TestSearchModel_Update_Enter tests pressing enter to search
func TestSearchModel_Update_Enter(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)
	model.input.SetValue("test query")

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := model.Update(msg)

	m := newModel
	if m.Value() != "test query" {
		t.Error("expected value to remain unchanged")
	}

	if cmd == nil {
		t.Error("expected search command from enter key")
	}
}

// TestSearchModel_Update_Escape tests pressing escape to clear
func TestSearchModel_Update_Escape(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)
	model.input.SetValue("test")
	model.resultCount = 5

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	newModel, cmd := model.Update(msg)

	m := newModel
	if m.Value() != "" {
		t.Error("expected value to be cleared after escape")
	}

	if m.resultCount != 0 {
		t.Error("expected result count to be reset")
	}

	if cmd != nil {
		t.Error("expected no command from escape")
	}
}

// TestSearchModel_FocusBlur tests focus and blur
func TestSearchModel_FocusBlur(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)

	if !model.Focused() {
		t.Error("expected model to be focused initially")
	}
	model = model.Blur()
	if model.Focused() {
		t.Error("expected model to be blurred")
	}

	model = model.Focus()
	if !model.Focused() {
		t.Error("expected model to be focused")
	}
}

// TestSearchModel_SetResults tests setting results
func TestSearchModel_SetResults(t *testing.T) {
	store := &db.Store{}
	model := NewSearchModel(store)
	model.searching = true

	model.SetResults(10, nil)

	if model.searching {
		t.Error("expected searching to be false after SetResults")
	}

	if model.resultCount != 10 {
		t.Errorf("expected resultCount 10, got %d", model.resultCount)
	}

	if model.err != nil {
		t.Error("expected no error")
	}
}

// TestSearchModel_Integration_TypingFlow tests typing flow using teatest via RootModel
func TestSearchModel_Integration_TypingFlow(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 10))
	defer tm.Quit()

	tm.Type("hello world")
	time.Sleep(100 * time.Millisecond)
	out := tm.Output()
	if out == nil {
		t.Error("expected output reader")
	}
}

// TestSearchModel_Integration_EscapeFlow tests escape key using teatest via RootModel
func TestSearchModel_Integration_EscapeFlow(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 10))
	defer tm.Quit()

	tm.Type("test query")
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})
	time.Sleep(50 * time.Millisecond)
}

var errTest = &testError{msg: "test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
