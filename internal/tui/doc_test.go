package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stormlightlabs/documango/internal/db"
)

// TestNewDocModel verifies doc model initialization
func TestNewDocModel(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	if model.store != store {
		t.Error("expected store to be set")
	}

	if model.loading {
		t.Error("expected loading to be false initially")
	}

	if model.content != "" {
		t.Error("expected empty content initially")
	}
}

// TestDocModel_Init verifies init returns spinner tick
func TestDocModel_Init(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	cmd := model.Init()
	if cmd == nil {
		t.Error("expected Init to return a command (spinner tick)")
	}
}

// TestDocModel_View_Empty tests view with no content
func TestDocModel_View_Empty(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	if !strings.Contains(view, "No document loaded") {
		t.Error("expected view to show empty state")
	}
}

// TestDocModel_View_Loading tests view while loading
func TestDocModel_View_Loading(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.loading = true

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view while loading")
	}

	if !strings.Contains(view, "Loading") {
		t.Error("expected view to show loading indicator")
	}
}

// TestDocModel_View_WithError tests view with error
func TestDocModel_View_WithError(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.err = errTest

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view with error")
	}

	if !strings.Contains(view, "Error") {
		t.Error("expected view to show error message")
	}
}

// TestDocModel_View_WithContent tests view with content
func TestDocModel_View_WithContent(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.content = "# Test Document\n\nThis is test content."
	model.path = "test/path.md"

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view with content")
	}
}

// TestDocModel_Update_LoadDoc tests loading a document
func TestDocModel_Update_LoadDoc(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	msg := loadDocMsg{docID: 1}
	newModel, cmd := model.Update(msg)

	m := newModel
	if m.docID != 1 {
		t.Errorf("expected docID 1, got %d", m.docID)
	}

	if !m.loading {
		t.Error("expected loading to be true after loadDocMsg")
	}

	if cmd == nil {
		t.Error("expected command to load document")
	}
}

// TestDocModel_Update_DocLoaded tests handling loaded document
func TestDocModel_Update_DocLoaded(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.loading = true

	msg := docLoadedMsg{
		content: "# Test\n\nContent",
		path:    "test.md",
		links:   []Link{{index: 1, target: "link1", text: "Link 1"}},
		err:     nil,
	}

	newModel, cmd := model.Update(msg)

	m := newModel
	if m.loading {
		t.Error("expected loading to be false after docLoadedMsg")
	}

	if m.content != "# Test\n\nContent" {
		t.Errorf("expected content set, got %q", m.content)
	}

	if m.path != "test.md" {
		t.Errorf("expected path 'test.md', got %q", m.path)
	}

	if len(m.links) != 1 {
		t.Errorf("expected 1 link, got %d", len(m.links))
	}

	if cmd != nil {
		t.Error("expected no command from docLoadedMsg")
	}
}

// TestDocModel_Update_DocLoadedError tests handling load error
func TestDocModel_Update_DocLoadedError(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.loading = true

	msg := docLoadedMsg{
		content: "",
		path:    "",
		links:   nil,
		err:     errTest,
	}

	newModel, cmd := model.Update(msg)

	m := newModel
	if m.loading {
		t.Error("expected loading to be false after error")
	}

	if m.err == nil {
		t.Error("expected error to be set")
	}

	if cmd != nil {
		t.Error("expected no command from docLoadedMsg with error")
	}
}

// TestDocModel_Update_ScrollKeys tests scroll navigation keys
func TestDocModel_Update_ScrollKeys(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, cmd := model.Update(msg)
	if cmd != nil {
		t.Error("expected no command from scroll key")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from scroll key")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from scroll key")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from scroll key")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from scroll key")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	newModel, cmd = newModel.Update(msg)
	if cmd != nil {
		t.Error("expected no command from scroll key")
	}

	_ = newModel
}

// TestDocModel_Update_LinkKeys tests link activation keys
func TestDocModel_Update_LinkKeys(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.links = []Link{
		{index: 1, target: "link1", text: "Link 1"},
		{index: 2, target: "link2", text: "Link 2"},
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	newModel, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected command from link key")
	}

	if cmd != nil {
		result := cmd()
		if linkMsg, ok := result.(docLinkMsg); ok {
			if linkMsg.target != "link1" {
				t.Errorf("expected target 'link1', got %q", linkMsg.target)
			}
		} else {
			t.Error("expected docLinkMsg from link key")
		}
	}

	_ = newModel
}

// TestDocModel_Update_Escape tests escape key
func TestDocModel_Update_Escape(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	newModel, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected command from escape key")
	}

	if cmd != nil {
		result := cmd()
		if _, ok := result.(backToListMsg); !ok {
			t.Error("expected backToListMsg from escape key")
		}
	}

	_ = newModel
}

// TestDocModel_Update_FocusSearch tests focus search key
func TestDocModel_Update_FocusSearch(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	newModel, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected command from / key")
	}

	if cmd != nil {
		result := cmd()
		if _, ok := result.(focusSearchMsg); !ok {
			t.Error("expected focusSearchMsg from / key")
		}
	}

	_ = newModel
}

// TestDocModel_Update_WindowSize tests window resize
func TestDocModel_Update_WindowSize(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, cmd := model.Update(msg)

	if cmd != nil {
		t.Error("expected no command from window size message")
	}

	m := newModel
	if m.viewport.Width != 100 {
		t.Errorf("expected viewport width 100, got %d", m.viewport.Width)
	}

	if m.viewport.Height != 47 {
		t.Errorf("expected viewport height 47, got %d", m.viewport.Height)
	}
}

// TestDocModel_Path tests getting the document path
func TestDocModel_Path(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.path = "test/document.md"

	if model.Path() != "test/document.md" {
		t.Errorf("expected path 'test/document.md', got %q", model.Path())
	}
}

// TestDocModel_DocID tests getting the document ID
func TestDocModel_DocID(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.docID = 42

	if model.DocID() != 42 {
		t.Errorf("expected DocID 42, got %d", model.DocID())
	}
}

// TestDocModel_LoadDocument tests the LoadDocument command creator
func TestDocModel_LoadDocument(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	cmd := model.LoadDocument(123)
	if cmd == nil {
		t.Error("expected LoadDocument to return a command")
	}

	if cmd != nil {
		msg := cmd()
		if loadMsg, ok := msg.(loadDocMsg); ok {
			if loadMsg.docID != 123 {
				t.Errorf("expected docID 123, got %d", loadMsg.docID)
			}
		} else {
			t.Error("expected loadDocMsg from LoadDocument")
		}
	}
}

// TestNewLink tests creating a new link
func TestNewLink(t *testing.T) {
	link := NewLink(1, "target", "text")

	if link.index != 1 {
		t.Errorf("expected index 1, got %d", link.index)
	}

	if link.target != "target" {
		t.Errorf("expected target 'target', got %q", link.target)
	}

	if link.text != "text" {
		t.Errorf("expected text 'text', got %q", link.text)
	}
}

// TestDocModel_ExtractLinks tests link extraction from markdown
func TestDocModel_ExtractLinks(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	content := `# Test Document

This is a [link one](path/to/one) and another [link two](path/to/two).

Here's [link three](path/to/three).`

	links := model.extractLinks(content)

	if len(links) != 3 {
		t.Errorf("expected 3 links, got %d", len(links))
	}

	if links[0].index != 1 {
		t.Errorf("expected first link index 1, got %d", links[0].index)
	}

	if links[0].target != "path/to/one" {
		t.Errorf("expected first link target 'path/to/one', got %q", links[0].target)
	}

	if links[0].text != "link one" {
		t.Errorf("expected first link text 'link one', got %q", links[0].text)
	}
}

// TestDocModel_ExtractLinks_NoLinks tests extraction with no links
func TestDocModel_ExtractLinks_NoLinks(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	content := `# Test Document

This has no links.`

	links := model.extractLinks(content)

	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

// TestDocModel_RenderHeader tests header rendering
func TestDocModel_RenderHeader(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	header := model.renderHeader()
	if header != "" {
		t.Errorf("expected empty header with no path, got %q", header)
	}

	model.path = "test/path.md"
	header = model.renderHeader()
	if header == "" {
		t.Error("expected non-empty header with path")
	}

	if !strings.Contains(header, "test/path.md") {
		t.Error("expected header to contain path")
	}
}

// TestDocModel_RenderFooter tests footer rendering
func TestDocModel_RenderFooter(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	footer := model.renderFooter()
	if footer != "" {
		t.Errorf("expected empty footer with no links, got %q", footer)
	}

	model.links = []Link{
		{index: 1, target: "link1", text: "Link 1"},
		{index: 2, target: "link2", text: "Link 2"},
	}
	footer = model.renderFooter()
	if footer == "" {
		t.Error("expected non-empty footer with links")
	}

	if !strings.Contains(footer, "Links:") {
		t.Error("expected footer to contain 'Links:'")
	}

	if !strings.Contains(footer, "[1]Link 1") {
		t.Error("expected footer to contain first link")
	}
}

// TestDocModel_RenderMarkdown tests markdown rendering (basic)
func TestDocModel_RenderMarkdown(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)

	markdown := "# Hello\n\nThis is a test."
	rendered, err := model.renderMarkdown(markdown)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if rendered == "" {
		t.Error("expected non-empty rendered content")
	}
}

// TestDocMessages tests that message types can be created
func TestDocMessages(t *testing.T) {
	loadMsg := loadDocMsg{docID: 1}
	if loadMsg.docID != 1 {
		t.Error("unexpected docID")
	}

	loadedMsg := docLoadedMsg{content: "content"}
	if loadedMsg.content != "content" {
		t.Error("unexpected content")
	}

	linkMsg := docLinkMsg{target: "target"}
	if linkMsg.target != "target" {
		t.Error("unexpected target")
	}

	_ = backToListMsg{}
}

// TestDocModel_LinkKey_InvalidIndex tests pressing invalid link keys
func TestDocModel_LinkKey_InvalidIndex(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.links = []Link{
		{index: 1, target: "link1", text: "Link 1"},
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}}
	newModel, cmd := model.Update(msg)

	if cmd != nil {
		t.Error("expected no command for invalid link index")
	}

	_ = newModel
}

// TestDocModel_LinkKey_Zero tests pressing '0' (not a valid link)
func TestDocModel_LinkKey_Zero(t *testing.T) {
	store := &db.Store{}
	model := NewDocModel(store)
	model.links = []Link{
		{index: 1, target: "link1", text: "Link 1"},
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}}
	newModel, cmd := model.Update(msg)

	if cmd != nil {
		t.Error("expected no command for '0' key")
	}

	_ = newModel
}
