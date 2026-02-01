package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tab represents a single open document tab.
type Tab struct {
	Title   string
	DocID   int64
	Content string
}

func newTab(id int64, t, c string) Tab {
	return Tab{Title: t, DocID: id, Content: c}
}

// TabBar manages the tab stack and rendering.
type TabBar struct {
	tabs      []Tab
	activeIdx int
	maxTabs   int
	width     int
}

// NewTabBar creates a new tab bar with default settings.
func NewTabBar() TabBar {
	return TabBar{
		tabs:    make([]Tab, 0),
		maxTabs: 10,
		width:   80,
	}
}

// AddTab adds a new tab to the stack.
// Returns false if max tabs reached.
func (tb *TabBar) AddTab(title string, docID int64, content string) bool {
	if len(tb.tabs) >= tb.maxTabs {
		return false
	}

	for i, tab := range tb.tabs {
		if tab.DocID == docID {
			tb.activeIdx = i
			return true
		}
	}

	tb.tabs = append(tb.tabs, newTab(docID, title, content))
	tb.activeIdx = len(tb.tabs) - 1
	return true
}

// CloseTab closes the current tab and returns to the previous one.
// Returns false if no tabs to close.
func (tb *TabBar) CloseTab() bool {
	if len(tb.tabs) == 0 {
		return false
	}

	tb.tabs = append(tb.tabs[:tb.activeIdx], tb.tabs[tb.activeIdx+1:]...)

	if tb.activeIdx >= len(tb.tabs) {
		tb.activeIdx = len(tb.tabs) - 1
	}
	if tb.activeIdx < 0 {
		tb.activeIdx = 0
	}

	return len(tb.tabs) > 0
}

// NextTab cycles to the next tab.
func (tb *TabBar) NextTab() {
	if len(tb.tabs) == 0 {
		return
	}
	tb.activeIdx = (tb.activeIdx + 1) % len(tb.tabs)
}

// PrevTab cycles to the previous tab.
func (tb *TabBar) PrevTab() {
	if len(tb.tabs) == 0 {
		return
	}
	tb.activeIdx = (tb.activeIdx - 1 + len(tb.tabs)) % len(tb.tabs)
}

// ActiveTab returns the currently active tab.
func (tb TabBar) ActiveTab() (Tab, bool) {
	if tb.activeIdx < 0 || tb.activeIdx >= len(tb.tabs) {
		return Tab{}, false
	}
	return tb.tabs[tb.activeIdx], true
}

// SetActiveIdx sets the active tab index.
func (tb *TabBar) SetActiveIdx(idx int) {
	if idx >= 0 && idx < len(tb.tabs) {
		tb.activeIdx = idx
	}
}

// HasTabs returns true if there are open tabs.
func (tb TabBar) HasTabs() bool {
	return len(tb.tabs) > 0
}

// TabCount returns the number of open tabs.
func (tb TabBar) TabCount() int {
	return len(tb.tabs)
}

// SetWidth sets the tab bar width for rendering.
func (tb *TabBar) SetWidth(w int) {
	tb.width = w
}

// Render renders the tab bar as a string.
func (tb TabBar) Render() string {
	if len(tb.tabs) == 0 {
		return ""
	}

	var parts []string
	availableWidth := tb.width - 4

	for i, tab := range tb.tabs {
		title := tab.Title
		maxTitleLen := 20
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-3] + "..."
		}

		if i == tb.activeIdx {
			title = title + "*"
		}

		if i == tb.activeIdx {
			parts = append(parts, activeTabStyle.Render(" "+title+" "))
		} else {
			parts = append(parts, inactiveTabStyle.Render(" "+title+" "))
		}

		currentLen := 0
		for _, p := range parts {
			currentLen += lipgloss.Width(p)
		}
		if currentLen > availableWidth && i < len(tb.tabs)-1 {
			parts = append(parts, dimStyle.Render(" ..."))
			break
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// GetTabDocID returns the DocID for a given tab index.
func (tb TabBar) GetTabDocID(idx int) (int64, bool) {
	if idx < 0 || idx >= len(tb.tabs) {
		return 0, false
	}
	return tb.tabs[idx].DocID, true
}

// UpdateActiveTabContent updates the content of the active tab.
func (tb *TabBar) UpdateActiveTabContent(content string) {
	if tb.activeIdx >= 0 && tb.activeIdx < len(tb.tabs) {
		tb.tabs[tb.activeIdx].Content = content
	}
}

// TabLimitReached returns true if max tabs reached.
func (tb TabBar) TabLimitReached() bool {
	return len(tb.tabs) >= tb.maxTabs
}

// tabSwitchMsg is sent when switching tabs.
type tabSwitchMsg struct {
	direction int // 1 for next, -1 for prev
}

// tabCloseMsg is sent when closing a tab.
type tabCloseMsg struct{}

// openLinkMsg is sent when opening a link in a new tab.
type openLinkMsg struct {
	target string
	docID  int64
}

// truncateTitle truncates a title to fit within maxLen.
func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}
	return title[:maxLen-3] + "..."
}

// formatTabTitle formats a tab title with index for display.
func formatTabTitle(idx int, title string, isActive bool) string {
	prefix := fmt.Sprintf("%d:", idx+1)
	if isActive {
		return prefix + title + "*"
	}
	return prefix + title
}
