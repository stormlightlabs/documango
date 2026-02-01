package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/shared"
)

// searchTickMsg is sent when the debounce timer expires.
type searchTickMsg struct{ query string }

// searchResultsMsg is sent when search results are ready.
type searchResultsMsg struct {
	results []db.SearchResult
	query   string
}

// searchErrMsg is sent when a search fails.
type searchErrMsg struct{ err error }

// SearchModel is the search input component.
type SearchModel struct {
	input       textinput.Model
	spinner     spinner.Model
	store       *db.Store
	debounce    time.Duration
	lastQuery   string
	resultCount int
	searching   bool
	err         error
}

// NewSearchModel creates a new search model.
func NewSearchModel(store *db.Store) SearchModel {
	input := textinput.New()
	input.Placeholder = "Search documentation..."
	input.Focus()
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	d := 150 * time.Millisecond
	return SearchModel{input: input, spinner: s, store: store, debounce: d}
}

// Init returns the initial command.
func (m SearchModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// Update handles messages.
func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.input.Value() != "" {
				return m, m.performSearch(m.input.Value())
			}
		case "esc":
			m.input.Reset()
			m.resultCount = 0
			m.err = nil
			return m, nil
		}

		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)

		query := m.input.Value()
		if query != m.lastQuery && len(query) >= 2 {
			m.lastQuery = query
			return m, tea.Sequence(cmd, m.startDebounce(query))
		}

		return m, cmd

	case searchTickMsg:
		if msg.query == m.input.Value() {
			return m, m.performSearch(msg.query)
		}
		return m, nil

	case spinner.TickMsg:
		if m.searching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// View renders the search input.
func (m SearchModel) View() string {
	var status string
	if m.err != nil {
		status = errorStyle.Render(" Search failed: " + m.err.Error())
	} else if m.searching {
		status = m.spinner.View() + " Searching..."
	} else if m.resultCount > 0 {
		status = accentStyle.Render(" " + shared.Itoa(m.resultCount) + " results")
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, searchInputStyle.Render(m.input.View()), status)
}

// Value returns the current search query.
func (m SearchModel) Value() string {
	return m.input.Value()
}

// Focused returns whether the input is focused.
func (m SearchModel) Focused() bool {
	return m.input.Focused()
}

// Focus sets focus on the input.
func (m SearchModel) Focus() SearchModel {
	m.input.Focus()
	return m
}

// Blur removes focus from the input.
func (m SearchModel) Blur() SearchModel {
	m.input.Blur()
	return m
}

// startDebounce starts the debounce timer.
func (m SearchModel) startDebounce(query string) tea.Cmd {
	return tea.Tick(m.debounce, func(_ time.Time) tea.Msg {
		return searchTickMsg{query: query}
	})
}

// performSearch executes the search query.
func (m SearchModel) performSearch(query string) tea.Cmd {
	m.searching = true
	return func() tea.Msg {
		ctx := context.Background()
		results, err := m.store.Search(ctx, query, 100)
		if err != nil {
			return searchErrMsg{err: err}
		}
		return searchResultsMsg{results: results, query: query}
	}
}

// SetResults updates the model with search results.
func (m *SearchModel) SetResults(count int, err error) {
	m.searching = false
	m.resultCount = count
	m.err = err
}
