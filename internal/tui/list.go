package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/stormlightlabs/documango/internal/db"
)

// listSelectMsg is sent when the user presses Enter on a result.
type listSelectMsg struct {
	result db.SearchResult
}

// ResultItem wraps db.SearchResult for display in the list.
type ResultItem struct {
	result db.SearchResult
}

// NewResultItem creates a new list item from a search result.
func NewResultItem(r db.SearchResult) ResultItem {
	return ResultItem{result: r}
}

// Result returns the underlying search result.
func (i ResultItem) Result() db.SearchResult {
	return i.result
}

// FilterValue implements list.Item.
func (i ResultItem) FilterValue() string {
	return i.result.Name
}

// ResultDelegate defines how items are rendered in the list.
type ResultDelegate struct{}

// NewResultDelegate creates a new result delegate.
func NewResultDelegate() ResultDelegate {
	return ResultDelegate{}
}

// Height implements list.ItemDelegate.
func (d ResultDelegate) Height() int {
	return 2
}

// Spacing implements list.ItemDelegate.
func (d ResultDelegate) Spacing() int {
	return 1
}

// Update implements list.ItemDelegate.
func (d ResultDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render implements list.ItemDelegate.
func (d ResultDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(ResultItem)
	if !ok {
		return
	}

	results := i.result
	var (
		name, typeStr string
		isSelected    = index == m.Index()
	)

	if isSelected {
		name = selectedNameStyle.Render(results.Name)
		typeStr = selectedTypeStyle.Render(results.Type)
	} else {
		name = nameStyle.Render(results.Name)
		typeStr = typeStyle.Render(results.Type)
	}

	fmt.Fprintf(w, "%s\n%s", name, typeStr)
}

// ListModel wraps bubbles/list for search result navigation.
type ListModel struct {
	list     list.Model
	results  []db.SearchResult
	selected *db.SearchResult
}

// NewListModel creates a new list model.
func NewListModel() ListModel {
	delegate := NewResultDelegate()

	l := list.New(nil, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(true)
	l.SetShowFilter(false)
	l.DisableQuitKeybindings()

	l.KeyMap.NextPage.SetKeys("ctrl+d", "pgdown")
	l.KeyMap.PrevPage.SetKeys("ctrl+u", "pgup")
	l.KeyMap.GoToStart.SetKeys("g", "home")
	l.KeyMap.GoToEnd.SetKeys("G", "end")

	return ListModel{list: l}
}

// Init returns the initial command.
func (m ListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.list.Index() >= 0 && m.list.SelectedItem() != nil {
				item, ok := m.list.SelectedItem().(ResultItem)
				if ok {
					m.selected = &item.result
					return m, func() tea.Msg {
						return listSelectMsg{result: item.result}
					}
				}
			}
		case "/":
			return m, func() tea.Msg {
				return focusSearchMsg{}
			}
		case "j", "down":
			m.list.CursorDown()
			return m, nil
		case "k", "up":
			m.list.CursorUp()
			return m, nil
		case "G":
			if len(m.list.Items()) > 0 {
				m.list.Select(len(m.list.Items()) - 1)
			}
			return m, nil
		case "g":
			if len(m.list.Items()) > 0 {
				m.list.Select(0)
			}
			return m, nil
		}

	case searchResultsMsg:
		m.SetResults(msg.results)
		return m, nil
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the list.
func (m ListModel) View() string {
	if len(m.list.Items()) == 0 {
		return emptyStateStyle.Render("No results found. Try a different search term.")
	}

	return m.list.View()
}

// SetResults updates the list with new search results.
func (m *ListModel) SetResults(results []db.SearchResult) {
	m.results = results
	items := make([]list.Item, len(results))
	for i, r := range results {
		items[i] = NewResultItem(r)
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}

// Selected returns the currently selected result.
func (m ListModel) Selected() *db.SearchResult {
	return m.selected
}

// ClearSelection clears the selected result.
func (m *ListModel) ClearSelection() {
	m.selected = nil
}

// Focus sets focus on the list.
func (m ListModel) Focus() ListModel {
	return m
}

// Blur removes focus from the list.
func (m ListModel) Blur() ListModel {
	return m
}

// Focused returns whether the list is focused.
func (m ListModel) Focused() bool {
	return true
}

// SetSize sets the width and height of the list.
func (m *ListModel) SetSize(w, h int) {
	m.list.SetWidth(w)
	m.list.SetHeight(h)
}

// focusSearchMsg is sent when user presses "/" to return to search.
type focusSearchMsg struct{}
