package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
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
	showHelp bool
	search   SearchModel
	list     ListModel
	doc      DocModel
	tabs     TabBar
	help     help.Model
	keys     keyBindings
}

// NewRootModel creates a new root application model.
func NewRootModel(store *db.Store) RootModel {
	h := help.New()
	h.ShowAll = true
	return RootModel{
		store:  store,
		mode:   modeSearch,
		search: NewSearchModel(store),
		list:   NewListModel(),
		doc:    NewDocModel(store),
		tabs:   NewTabBar(),
		help:   h,
		keys:   newKeyBindings(),
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
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "ctrl+tab":
			if m.tabs.HasTabs() {
				m.tabs.NextTab()
				if tab, ok := m.tabs.ActiveTab(); ok {
					m.doc = NewDocModel(m.store)
					return m, m.doc.LoadDocument(tab.DocID)
				}
			}
			return m, nil
		case "ctrl+w":
			if m.tabs.HasTabs() {
				m.tabs.CloseTab()
				if tab, ok := m.tabs.ActiveTab(); ok {
					m.doc = NewDocModel(m.store)
					return m, m.doc.LoadDocument(tab.DocID)
				} else {
					m.mode = modeList
					m.list = m.list.Focus()
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
		_, _ = m.doc.Update(msg)
		m.tabs.SetWidth(msg.Width)

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
		m.tabs.AddTab(msg.result.Name, msg.result.DocID, "")
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
		if !m.tabs.TabLimitReached() {
			ctx := context.Background()
			doc, err := m.store.ReadDocument(ctx, msg.target)
			if err == nil {
				results, err := m.store.Search(ctx, doc.Path, 1)
				if err == nil && len(results) > 0 {
					m.tabs.AddTab(results[0].Name, results[0].DocID, "")
					m.doc = NewDocModel(m.store)
					return m, m.doc.LoadDocument(results[0].DocID)
				}
			}
		}
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

	if m.showHelp {
		return m.renderHelpView()
	}

	var helpText string

	switch m.mode {
	case modeSearch:
		searchView := m.search.View()
		helpText = m.help.View(m.keys)
		return lipgloss.JoinVertical(lipgloss.Left, searchView, "", helpText)
	case modeList:
		searchView := m.search.View()
		helpText = m.help.View(m.keys)
		return lipgloss.JoinVertical(lipgloss.Left, searchView, "", m.list.View(), "", helpText)
	case modeDoc:
		var parts []string
		if m.tabs.HasTabs() {
			parts = append(parts, m.tabs.Render())
		}
		parts = append(parts, m.doc.View())
		helpText := m.help.View(m.keys)
		parts = append(parts, "", helpText)
		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	default:
		return ""
	}
}

// renderHelpView renders the full help overlay.
func (m RootModel) renderHelpView() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Keyboard Shortcuts"),
		"",
		m.help.View(m.keys),
		"",
		dimStyle.Render("Press ? to close help"),
	)
}
