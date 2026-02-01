package tui

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stormlightlabs/documango/internal/db"
)

// mockStore is a minimal mock for testing TUI components
type mockStore struct{}

func (m *mockStore) Search(ctx context.Context, query string, limit int) ([]db.SearchResult, error) {
	return []db.SearchResult{{Name: "Test Result", Type: "function", DocID: 1}}, nil
}

func (m *mockStore) ReadDocumentByID(ctx context.Context, id int64) (db.Document, error) {
	return db.Document{Path: "test/path", Body: []byte("# Test Document\n\nThis is a test.")}, nil
}

// TestRootModel_Init verifies the root model initializes correctly
func TestRootModel_Init(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	cmd := model.Init()
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

// TestRootModel_View_SearchMode tests the view in search mode
func TestRootModel_View_SearchMode(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	model.mode = modeSearch

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view in search mode")
	}
}

// TestRootModel_View_ListMode tests the view in list mode
func TestRootModel_View_ListMode(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	model.mode = modeList
	view := model.View()
	if view == "" {
		t.Error("expected non-empty view in list mode")
	}
}

// TestRootModel_View_DocMode tests the view in doc mode
func TestRootModel_View_DocMode(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	model.mode = modeDoc
	view := model.View()
	if view == "" {
		t.Error("expected non-empty view in doc mode")
	}
}

// TestRootModel_View_HelpMode tests the help overlay view
func TestRootModel_View_HelpMode(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	model.showHelp = true

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view in help mode")
	}

	if !bytes.Contains([]byte(view), []byte("Keyboard")) {
		t.Error("expected help view to contain 'Keyboard'")
	}
}

// TestRootModel_View_Quitting tests the quitting view
func TestRootModel_View_Quitting(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	model.quitting = true
	view := model.View()
	if view != "Goodbye!\n" {
		t.Errorf("expected 'Goodbye!\\n', got %q", view)
	}
}

// TestRootModel_Update_Quit tests quitting the application
func TestRootModel_Update_Quit(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := model.Update(msg)
	m := newModel.(RootModel)
	if !m.quitting {
		t.Error("expected quitting to be true after ctrl+c")
	}

	if cmd == nil {
		t.Error("expected quit command")
	}
}

// TestRootModel_Update_ToggleHelp tests toggling help overlay
func TestRootModel_Update_ToggleHelp(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newModel, cmd := model.Update(msg)
	m := newModel.(RootModel)
	if !m.showHelp {
		t.Error("expected showHelp to be true after pressing ?")
	}

	if cmd != nil {
		t.Error("expected no command when toggling help")
	}

	newModel, _ = m.Update(msg)
	m = newModel.(RootModel)
	if m.showHelp {
		t.Error("expected showHelp to be false after pressing ? again")
	}
}

// TestRootModel_Update_WindowSize tests window resize handling
func TestRootModel_Update_WindowSize(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, cmd := model.Update(msg)
	m := newModel.(RootModel)
	if cmd != nil {
		t.Error("expected no command from window size message")
	}

	_ = m
}

// TestRootModel_Update_FocusSearch tests focusing search
func TestRootModel_Update_FocusSearch(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	model.mode = modeList
	msg := focusSearchMsg{}
	newModel, cmd := model.Update(msg)

	m := newModel.(RootModel)
	if m.mode != modeSearch {
		t.Error("expected mode to be modeSearch after focusSearchMsg")
	}

	if !m.search.Focused() {
		t.Error("expected search to be focused")
	}

	if cmd != nil {
		t.Error("expected no command from focusSearchMsg")
	}
}

// TestRootModel_Update_BackToList tests returning to list from doc
func TestRootModel_Update_BackToList(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	model.mode = modeDoc

	msg := backToListMsg{}
	newModel, cmd := model.Update(msg)

	m := newModel.(RootModel)
	if m.mode != modeList {
		t.Error("expected mode to be modeList after backToListMsg")
	}

	if cmd != nil {
		t.Error("expected no command from backToListMsg")
	}
}

// TestRootModel_Integration_SearchFlow tests the full search flow using teatest
func TestRootModel_Integration_SearchFlow(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(100, 50))
	defer tm.Quit()

	tm.Send(tea.WindowSizeMsg{Width: 100, Height: 50})
	time.Sleep(50 * time.Millisecond)

	out := tm.Output()
	if out == nil {
		t.Error("expected output reader")
	}
}

// TestRootModel_Integration_QuitFlow tests quitting using teatest
func TestRootModel_Integration_QuitFlow(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	out := tm.FinalOutput(t)
	if out == nil {
		t.Error("expected final output")
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, out); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("Goodbye")) {
		t.Error("expected output to contain 'Goodbye'")
	}
}

// TestRootModel_Integration_HelpToggle tests toggling help using teatest
func TestRootModel_Integration_HelpToggle(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 40))
	defer tm.Quit()

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	time.Sleep(100 * time.Millisecond)
	out := tm.Output()
	if out == nil {
		t.Error("expected output reader")
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	time.Sleep(100 * time.Millisecond)
}

// TestRootModel_Integration_TabNavigation tests tab navigation using teatest
func TestRootModel_Integration_TabNavigation(t *testing.T) {
	// TODO: create db for this
	t.Skip("Skipping integration test that requires database store")
}

// TestRootModel_FinalModel tests getting the final model
func TestRootModel_FinalModel(t *testing.T) {
	store := &db.Store{}
	model := NewRootModel(store)
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	finalModel := tm.FinalModel(t)
	if finalModel == nil {
		t.Error("expected final model")
	}

	m, ok := finalModel.(RootModel)
	if !ok {
		t.Fatal("expected RootModel type")
	}

	if !m.quitting {
		t.Error("expected model to be in quitting state")
	}
}
