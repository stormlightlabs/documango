package tui

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"

	"github.com/stormlightlabs/documango/internal/db"
	"github.com/stormlightlabs/documango/internal/shared"
)

// loadDocMsg is sent to trigger loading a document.
type loadDocMsg struct {
	docID int64
}

// docLoadedMsg is sent when a document is fetched and rendered.
type docLoadedMsg struct {
	content string
	path    string
	links   []Link
	err     error
}

// docLinkMsg is sent when a user activates a numbered link.
type docLinkMsg struct {
	target string
}

// Link represents a markdown link found in the document.
type Link struct {
	index  int
	target string
	text   string
}

func NewLink(i int, t, txt string) Link {
	return Link{index: i, target: t, text: txt}
}

// DocModel is the document viewer component.
type DocModel struct {
	store    *db.Store
	viewport viewport.Model
	spinner  spinner.Model
	content  string
	path     string
	links    []Link
	docID    int64
	loading  bool
	err      error
}

// NewDocModel creates a new document model without loading content.
func NewDocModel(s *db.Store) DocModel {
	v := viewport.New(0, 0)
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e"))
	return DocModel{store: s, viewport: v, spinner: sp, loading: false}
}

// Init returns the initial command.
func (m DocModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// LoadDocument returns a command to load a document by ID.
func (m DocModel) LoadDocument(docID int64) tea.Cmd {
	return func() tea.Msg {
		return loadDocMsg{docID: docID}
	}
}

// loadDocument fetches and renders the document.
func (m DocModel) loadDocument(docID int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		doc, err := m.store.ReadDocumentByID(ctx, docID)
		if err != nil {
			return docLoadedMsg{err: err}
		}

		path := doc.Path
		content, err := m.renderMarkdown(string(doc.Body))
		if err != nil {
			return docLoadedMsg{err: err}
		}

		links := m.extractLinks(string(doc.Body))
		return docLoadedMsg{content: content, path: path, links: links}
	}
}

// renderMarkdown renders markdown content using Glamour with a custom theme.
func (m DocModel) renderMarkdown(markdown string) (string, error) {
	theme := ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           shared.StringPtr("#fafafa"),
				BackgroundColor: shared.StringPtr("#0a0a0a"),
				BlockPrefix:     "",
				BlockSuffix:     "",
			},
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       shared.StringPtr("#22c55e"),
				Bold:        shared.BoolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:       shared.StringPtr("#22c55e"),
				Bold:        shared.BoolPtr(true),
				BlockSuffix: "\n",
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:       shared.StringPtr("#22c55e"),
				Bold:        shared.BoolPtr(true),
				BlockPrefix: "\n",
				BlockSuffix: "\n",
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:       shared.StringPtr("#22c55e"),
				Bold:        shared.BoolPtr(true),
				BlockPrefix: "\n",
				BlockSuffix: "\n",
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Bold: shared.BoolPtr(true), BlockPrefix: "\n"},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Bold: shared.BoolPtr(true), BlockPrefix: "\n"},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Bold: shared.BoolPtr(true), BlockPrefix: "\n"},
		},
		Text: ansi.StylePrimitive{Color: shared.StringPtr("#fafafa")},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:       shared.StringPtr("#737373"),
				Italic:      shared.BoolPtr(true),
				BlockPrefix: "> ",
			},
		},
		List: ansi.StyleList{LevelIndent: 2},
		Item: ansi.StylePrimitive{BlockPrefix: "• "},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color:           shared.StringPtr("#e5e5e5"),
					BackgroundColor: shared.StringPtr("#1f1f1f"),
					BlockPrefix:     "\n",
					BlockSuffix:     "\n",
				},
			},
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{BlockPrefix: "\n", BlockSuffix: "\n"},
		},
		Link:     ansi.StylePrimitive{Color: shared.StringPtr("#22c55e"), Underline: shared.BoolPtr(true)},
		LinkText: ansi.StylePrimitive{Color: shared.StringPtr("#22c55e"), Bold: shared.BoolPtr(true)},
	}

	r, err := glamour.NewTermRenderer(glamour.WithStyles(theme), glamour.WithWordWrap(80))
	if err != nil {
		return "", err
	}

	return r.Render(markdown)
}

// extractLinks finds markdown links in the content.
func (m DocModel) extractLinks(content string) []Link {
	re := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	matches := re.FindAllStringSubmatchIndex(content, -1)
	links := make([]Link, 0, len(matches))
	for i, match := range matches {
		if len(match) >= 6 {
			text := content[match[2]:match[3]]
			target := content[match[4]:match[5]]
			links = append(links, NewLink(i+1, target, text))
		}
	}

	return links
}

// Update handles messages.
func (m DocModel) Update(msg tea.Msg) (DocModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case loadDocMsg:
		m.docID = msg.docID
		m.loading = true
		m.err = nil
		return m, m.loadDocument(msg.docID)

	case docLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.content = msg.content
		m.path = msg.path
		m.links = msg.links
		m.viewport.SetContent(m.content)
		m.viewport.GotoTop()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k", "up":
			m.viewport.ScrollUp(1)
			return m, nil
		case "d":
			m.viewport.HalfPageDown()
			return m, nil
		case "u":
			m.viewport.HalfPageUp()
			return m, nil
		case "g":
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.viewport.GotoBottom()
			return m, nil
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '0')
			if idx > 0 && idx <= len(m.links) {
				return m, func() tea.Msg {
					return docLinkMsg{target: m.links[idx-1].target}
				}
			}
		case "esc":
			return m, func() tea.Msg {
				return backToListMsg{}
			}
		case "/":
			return m, func() tea.Msg {
				return focusSearchMsg{}
			}
		}

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the document viewer.
func (m DocModel) View() string {
	if m.err != nil {
		return errorStyle.Render("Error loading document: " + m.err.Error())
	}

	if m.loading {
		return lipgloss.JoinVertical(lipgloss.Left,
			m.spinner.View(),
			dimStyle.Render(" Loading document..."),
		)
	}

	if m.content == "" {
		return emptyStateStyle.Render("No document loaded.")
	}

	header := m.renderHeader()
	body := m.viewport.View()
	footer := m.renderFooter()
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// renderHeader renders the document header with path and back navigation.
func (m DocModel) renderHeader() string {
	if m.path == "" {
		return ""
	}
	title := docTitleStyle.Render(m.path)
	back := docBackStyle.Render(" Esc: back")
	return lipgloss.JoinHorizontal(lipgloss.Right, title, back)
}

// renderFooter renders the footer with links summary.
func (m DocModel) renderFooter() string {
	if len(m.links) == 0 {
		return ""
	}

	var linkParts []string
	for _, link := range m.links {
		linkParts = append(linkParts, fmt.Sprintf("[%d]%s", link.index, link.text))
	}

	linksText := strings.Join(linkParts, " ")
	return docLinksStyle.Render("Links: " + linksText)
}

// Path returns the document path.
func (m DocModel) Path() string {
	return m.path
}

// DocID returns the current document ID.
func (m DocModel) DocID() int64 {
	return m.docID
}

// backToListMsg is sent when user presses Esc to return to the list.
type backToListMsg struct{}
