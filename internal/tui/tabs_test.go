package tui

import (
	"strings"
	"testing"
)

// TestNewTabBar verifies tab bar initialization
func TestNewTabBar(t *testing.T) {
	tb := NewTabBar()

	if tb.maxTabs != 10 {
		t.Errorf("expected maxTabs 10, got %d", tb.maxTabs)
	}

	if tb.width != 80 {
		t.Errorf("expected width 80, got %d", tb.width)
	}

	if tb.HasTabs() {
		t.Error("expected no tabs initially")
	}

	if tb.TabCount() != 0 {
		t.Errorf("expected tab count 0, got %d", tb.TabCount())
	}
}

// TestTabBar_AddTab tests adding tabs
func TestTabBar_AddTab(t *testing.T) {
	tb := NewTabBar()

	if !tb.AddTab("Tab 1", 1, "content 1") {
		t.Error("expected AddTab to succeed")
	}

	if !tb.HasTabs() {
		t.Error("expected HasTabs to be true")
	}

	if tb.TabCount() != 1 {
		t.Errorf("expected tab count 1, got %d", tb.TabCount())
	}

	if !tb.AddTab("Tab 2", 2, "content 2") {
		t.Error("expected AddTab to succeed")
	}

	if tb.TabCount() != 2 {
		t.Errorf("expected tab count 2, got %d", tb.TabCount())
	}

	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 2 {
			t.Errorf("expected active tab DocID 2, got %d", tab.DocID)
		}
	} else {
		t.Error("expected to get active tab")
	}
}

// TestTabBar_AddTab_Duplicate tests adding duplicate tab
func TestTabBar_AddTab_Duplicate(t *testing.T) {
	tb := NewTabBar()

	tb.AddTab("Tab 1", 1, "content 1")
	tb.AddTab("Tab 2", 2, "content 2")

	if !tb.AddTab("Tab 1 Again", 1, "new content") {
		t.Error("expected AddTab to succeed for duplicate")
	}

	if tb.TabCount() != 2 {
		t.Errorf("expected tab count 2, got %d", tb.TabCount())
	}

	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 1 {
			t.Errorf("expected active tab DocID 1, got %d", tab.DocID)
		}
	}
}

// TestTabBar_AddTab_MaxTabs tests max tab limit
func TestTabBar_AddTab_MaxTabs(t *testing.T) {
	tb := NewTabBar()
	tb.maxTabs = 3

	tb.AddTab("Tab 1", 1, "")
	tb.AddTab("Tab 2", 2, "")
	tb.AddTab("Tab 3", 3, "")

	if tb.AddTab("Tab 4", 4, "") {
		t.Error("expected AddTab to fail at max tabs")
	}

	if tb.TabCount() != 3 {
		t.Errorf("expected tab count 3, got %d", tb.TabCount())
	}
}

// TestTabBar_CloseTab tests closing tabs
func TestTabBar_CloseTab(t *testing.T) {
	tb := NewTabBar()

	if tb.CloseTab() {
		t.Error("expected CloseTab to fail with no tabs")
	}

	tb.AddTab("Tab 1", 1, "")
	tb.AddTab("Tab 2", 2, "")
	tb.AddTab("Tab 3", 3, "")

	if !tb.CloseTab() {
		t.Error("expected CloseTab to succeed")
	}

	if tb.TabCount() != 2 {
		t.Errorf("expected tab count 2, got %d", tb.TabCount())
	}

	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 2 {
			t.Errorf("expected active tab DocID 2, got %d", tab.DocID)
		}
	}

	tb.CloseTab()
	tb.CloseTab()

	if tb.HasTabs() {
		t.Error("expected no tabs after closing all")
	}
}

// TestTabBar_NextTab tests cycling to next tab
func TestTabBar_NextTab(t *testing.T) {
	tb := NewTabBar()

	tb.NextTab()

	tb.AddTab("Tab 1", 1, "")
	tb.AddTab("Tab 2", 2, "")
	tb.AddTab("Tab 3", 3, "")

	tb.NextTab()
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 1 {
			t.Errorf("expected active tab DocID 1 after NextTab, got %d", tab.DocID)
		}
	}

	tb.NextTab()
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 2 {
			t.Errorf("expected active tab DocID 2 after NextTab, got %d", tab.DocID)
		}
	}
}

// TestTabBar_PrevTab tests cycling to previous tab
func TestTabBar_PrevTab(t *testing.T) {
	tb := NewTabBar()

	tb.PrevTab()

	tb.AddTab("Tab 1", 1, "")
	tb.AddTab("Tab 2", 2, "")
	tb.AddTab("Tab 3", 3, "")

	tb.PrevTab()
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 2 {
			t.Errorf("expected active tab DocID 2 after PrevTab, got %d", tab.DocID)
		}
	}

	tb.PrevTab()
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 1 {
			t.Errorf("expected active tab DocID 1 after PrevTab, got %d", tab.DocID)
		}
	}

	tb.PrevTab()
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 3 {
			t.Errorf("expected active tab DocID 3 after wrap-around, got %d", tab.DocID)
		}
	}
}

// TestTabBar_SetActiveIdx tests setting active index
func TestTabBar_SetActiveIdx(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("Tab 1", 1, "")
	tb.AddTab("Tab 2", 2, "")
	tb.AddTab("Tab 3", 3, "")

	tb.SetActiveIdx(0)
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 1 {
			t.Errorf("expected active tab DocID 1, got %d", tab.DocID)
		}
	}

	tb.SetActiveIdx(1)
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 2 {
			t.Errorf("expected active tab DocID 2, got %d", tab.DocID)
		}
	}

	tb.SetActiveIdx(10)
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 2 {
			t.Errorf("expected active tab DocID 2 (unchanged), got %d", tab.DocID)
		}
	}

	tb.SetActiveIdx(-1)
	if tab, ok := tb.ActiveTab(); ok {
		if tab.DocID != 2 {
			t.Errorf("expected active tab DocID 2 (unchanged), got %d", tab.DocID)
		}
	}
}

// TestTabBar_GetTabDocID tests getting tab DocID by index
func TestTabBar_GetTabDocID(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("Tab 1", 1, "")
	tb.AddTab("Tab 2", 2, "")

	docID, ok := tb.GetTabDocID(0)
	if !ok {
		t.Error("expected GetTabDocID to succeed for index 0")
	}
	if docID != 1 {
		t.Errorf("expected DocID 1, got %d", docID)
	}

	docID, ok = tb.GetTabDocID(1)
	if !ok {
		t.Error("expected GetTabDocID to succeed for index 1")
	}
	if docID != 2 {
		t.Errorf("expected DocID 2, got %d", docID)
	}

	_, ok = tb.GetTabDocID(10)
	if ok {
		t.Error("expected GetTabDocID to fail for invalid index")
	}
}

// TestTabBar_UpdateActiveTabContent tests updating tab content
func TestTabBar_UpdateActiveTabContent(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("Tab 1", 1, "old content")

	tb.UpdateActiveTabContent("new content")

	if tab, ok := tb.ActiveTab(); ok {
		if tab.Content != "new content" {
			t.Errorf("expected content 'new content', got %q", tab.Content)
		}
	}
}

// TestTabBar_TabLimitReached tests tab limit check
func TestTabBar_TabLimitReached(t *testing.T) {
	tb := NewTabBar()
	tb.maxTabs = 2

	if tb.TabLimitReached() {
		t.Error("expected TabLimitReached to be false initially")
	}

	tb.AddTab("Tab 1", 1, "")
	if tb.TabLimitReached() {
		t.Error("expected TabLimitReached to be false after 1 tab")
	}

	tb.AddTab("Tab 2", 2, "")
	if !tb.TabLimitReached() {
		t.Error("expected TabLimitReached to be true at max tabs")
	}
}

// TestTabBar_SetWidth tests setting width
func TestTabBar_SetWidth(t *testing.T) {
	tb := NewTabBar()
	tb.SetWidth(120)
	if tb.width != 120 {
		t.Errorf("expected width 120, got %d", tb.width)
	}
}

// TestTabBar_Render tests rendering
func TestTabBar_Render(t *testing.T) {
	tb := NewTabBar()

	rendered := tb.Render()
	if rendered != "" {
		t.Errorf("expected empty string for empty tab bar, got %q", rendered)
	}

	tb.AddTab("Tab 1", 1, "")
	rendered = tb.Render()
	if rendered == "" {
		t.Error("expected non-empty render with tabs")
	}

	if !strings.Contains(rendered, "Tab 1") {
		t.Error("expected render to contain tab title")
	}

	if !strings.Contains(rendered, "*") {
		t.Error("expected render to contain asterisk for active tab")
	}
}

// TestTabBar_Render_LongTitle tests rendering with long titles
func TestTabBar_Render_LongTitle(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("This is a very long title that should be truncated", 1, "")

	rendered := tb.Render()
	if !strings.Contains(rendered, "...") {
		t.Error("expected long title to be truncated with '...'")
	}
}

// TestTruncateTitle tests the truncateTitle helper
func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a long title", 10, "this is..."},
		{"exact", 5, "exact"},
		{"toolong", 3, "..."},
		{"ab", 1, "."},
	}

	for _, tt := range tests {
		got := truncateTitle(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateTitle(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

// TestFormatTabTitle tests the formatTabTitle helper
func TestFormatTabTitle(t *testing.T) {
	tests := []struct {
		idx      int
		title    string
		isActive bool
		want     string
	}{
		{0, "Tab", false, "1:Tab"},
		{0, "Tab", true, "1:Tab*"},
		{2, "Test", false, "3:Test"},
		{2, "Test", true, "3:Test*"},
	}

	for _, tt := range tests {
		got := formatTabTitle(tt.idx, tt.title, tt.isActive)
		if got != tt.want {
			t.Errorf("formatTabTitle(%d, %q, %v) = %q, want %q", tt.idx, tt.title, tt.isActive, got, tt.want)
		}
	}
}

// TestTabBar_ActiveTab_Empty tests ActiveTab with no tabs
func TestTabBar_ActiveTab_Empty(t *testing.T) {
	tb := NewTabBar()

	_, ok := tb.ActiveTab()
	if ok {
		t.Error("expected ActiveTab to return false with no tabs")
	}
}

// TestTabBar_CloseTab_LastTab tests closing the last tab
func TestTabBar_CloseTab_LastTab(t *testing.T) {
	tb := NewTabBar()
	tb.AddTab("Tab 1", 1, "")

	if tb.CloseTab() {
		t.Error("expected CloseTab to return false when closing last tab")
	}

	if tb.HasTabs() {
		t.Error("expected no tabs after closing last tab")
	}

	_, ok := tb.ActiveTab()
	if ok {
		t.Error("expected ActiveTab to return false after closing all tabs")
	}
}

// Ensure Tab implements the necessary interface methods
func TestTab_Struct(t *testing.T) {
	tab := Tab{
		Title:   "Test",
		DocID:   1,
		Content: "content",
	}

	if tab.Title != "Test" {
		t.Error("unexpected Title")
	}
	if tab.DocID != 1 {
		t.Error("unexpected DocID")
	}
	if tab.Content != "content" {
		t.Error("unexpected Content")
	}
}

// TestTabMessages tests that message types exist and can be created
func TestTabMessages(t *testing.T) {
	switchMsg := tabSwitchMsg{direction: 1}
	if switchMsg.direction != 1 {
		t.Error("unexpected direction")
	}

	_ = tabCloseMsg{}

	openMsg := openLinkMsg{target: "test", docID: 1}
	if openMsg.target != "test" {
		t.Error("unexpected target")
	}
	if openMsg.docID != 1 {
		t.Error("unexpected docID")
	}
}
