package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/stormlightlabs/documango/internal/db"
)

// appMode represents the current application state.
type appMode int

const (
	modeSearch appMode = iota
	modeList
	modeDoc
)

// RootModel is the top-level application model that orchestrates all components.
type RootModel struct {
	store    *db.Store
	mode     appMode
	quitting bool
	search   SearchModel
	list     ListModel
	doc      DocModel
}

// NewRootModel creates a new root application model.
func NewRootModel(store *db.Store) RootModel {
	return RootModel{
		store:  store,
		mode:   modeSearch,
		search: NewSearchModel(store),
		list:   NewListModel(),
		doc:    NewDocModel(store),
	}
}

// Init returns the initial command for startup.
func (m RootModel) Init() tea.Cmd {
	return m.search.Init()
}

// Update processes messages and returns the updated model.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			if m.mode != modeSearch || !m.search.Focused() {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
		_, _ = m.doc.Update(msg) // Update doc with new window size
		// Don't return cmd here, let it be handled in mode switch below

	case searchTickMsg:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd

	case searchResultsMsg:
		m.search.SetResults(len(msg.results), nil)
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if len(msg.results) > 0 && m.mode == modeSearch {
			m.mode = modeList
			m.search = m.search.Blur()
			m.list = m.list.Focus()
		}
		return m, cmd

	case searchErrMsg:
		m.search.SetResults(0, msg.err)
		return m, nil

	case listSelectMsg:
		m.mode = modeDoc
		m.list = m.list.Blur()
		m.doc = NewDocModel(m.store)
		return m, m.doc.LoadDocument(msg.result.DocID)

	case loadDocMsg:
		var cmd tea.Cmd
		m.doc, cmd = m.doc.Update(msg)
		return m, cmd

	case docLoadedMsg:
		var cmd tea.Cmd
		m.doc, cmd = m.doc.Update(msg)
		return m, cmd

	case backToListMsg:
		m.mode = modeList
		m.list = m.list.Focus()
		return m, nil

	case docLinkMsg:
		// TODO: Implement open new tab on link activation
		return m, nil

	case focusSearchMsg:
		m.mode = modeSearch
		m.search = m.search.Focus()
		m.list = m.list.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	switch m.mode {
	case modeSearch:
		m.search, cmd = m.search.Update(msg)
	case modeList:
		m.list, cmd = m.list.Update(msg)
	case modeDoc:
		m.doc, cmd = m.doc.Update(msg)
	}

	return m, cmd
}

// View renders the UI as a string.
func (m RootModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var help string

	switch m.mode {
	case modeSearch:
		searchView := m.search.View()
		help = helpStyle.Render("Ctrl+C: quit  Enter: search  Esc: clear")
		return lipgloss.JoinVertical(lipgloss.Left, searchView, "", help)
	case modeList:
		searchView := m.search.View()
		help = helpStyle.Render("j/k: nav  Enter: open  /: search  q: quit  Esc: back")
		return lipgloss.JoinVertical(lipgloss.Left, searchView, "", m.list.View(), "", help)
	case modeDoc:
		help = helpStyle.Render("j/k: scroll  g/G: top/bottom  d/u: half-page  Esc: back  /: search  1-9: link")
		return lipgloss.JoinVertical(lipgloss.Left, m.doc.View(), "", help)
	default:
		return ""
	}
}
