package web

import (
	"strings"
	"testing"
)

func TestNewMarkdownRenderer(t *testing.T) {
	renderer := NewMarkdownRenderer()
	if renderer == nil {
		t.Fatal("expected non-nil renderer")
	}
	if renderer.md == nil {
		t.Error("expected markdown instance to be initialized")
	}
}

func TestMarkdownRenderer_Render(t *testing.T) {
	renderer := NewMarkdownRenderer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple paragraph",
			input:    "Hello world",
			expected: []string{"<p>", "Hello world", "</p>"},
		},
		{
			name:     "heading",
			input:    "# Title",
			expected: []string{"<h1", "Title", "</h1>"},
		},
		{
			name:     "heading with anchor",
			input:    "## Section",
			expected: []string{"<h2", "Section", "</h2>"},
		},
		{
			name:     "bold text",
			input:    "**bold**",
			expected: []string{"<strong>", "bold", "</strong>"},
		},
		{
			name:     "italic text",
			input:    "*italic*",
			expected: []string{"<em>", "italic", "</em>"},
		},
		{
			name:     "code block",
			input:    "```go\nfunc main() {}\n```",
			expected: []string{"<div", "class=", "code-block"},
		},
		{
			name:     "inline code",
			input:    "`code`",
			expected: []string{"<code", "class=", "code-inline", "code", "</code>"},
		},
		{
			name:     "blockquote",
			input:    "> quote",
			expected: []string{"<blockquote", "class=", "callout", "quote", "</blockquote>"},
		},
		{
			name:     "table",
			input:    "| A | B |\n|---|---|\n| 1 | 2 |",
			expected: []string{"<table", "class=", "doc-table", "</table>"},
		},
		{
			name:     "link",
			input:    "[link](http://example.com)",
			expected: []string{"<a", "href=", "http://example.com", "link", "</a>"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render([]byte(tt.input))
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("expected output to contain %q, got:\n%s", exp, result)
				}
			}
		})
	}
}

func TestMarkdownRenderer_RenderWithTOC(t *testing.T) {
	renderer := NewMarkdownRenderer()

	input := `# Title

## Section 1

Content here.

### Subsection

More content.

## Section 2

Final content.`

	html, toc, err := renderer.RenderWithTOC([]byte(input))
	if err != nil {
		t.Fatalf("RenderWithTOC failed: %v", err)
	}

	if html == "" {
		t.Error("expected non-empty HTML")
	}

	if len(toc) == 0 {
		t.Error("expected non-empty TOC")
	}

	foundSections := make(map[string]bool)
	for _, item := range toc {
		foundSections[item.Text] = true
		if item.Level != 2 && item.Level != 3 {
			t.Errorf("expected level 2 or 3, got %d", item.Level)
		}
	}

	if !foundSections["Section 1"] {
		t.Error("expected TOC to contain 'Section 1'")
	}
	if !foundSections["Section 2"] {
		t.Error("expected TOC to contain 'Section 2'")
	}
	if !foundSections["Subsection"] {
		t.Error("expected TOC to contain 'Subsection'")
	}
	if foundSections["Title"] {
		t.Error("expected TOC to NOT contain h1 'Title'")
	}
}

func TestExtractTOC(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		expectedItems []struct {
			text  string
			level int
		}
	}{
		{
			name:          "no headings",
			input:         "Just some text",
			expectedCount: 0,
		},
		{
			name: "only h1",
			input: `# Title

Some content`,
			expectedCount: 0,
		},
		{
			name: "h2 and h3",
			input: `## Section 1

### Subsection

## Section 2`,
			expectedCount: 3,
			expectedItems: []struct {
				text  string
				level int
			}{
				{"Section 1", 2},
				{"Subsection", 3},
				{"Section 2", 2},
			},
		},
		{
			name: "mixed levels",
			input: `# Title
## Section
### Subsection
#### Deep
##### Deeper`,
			expectedCount: 2,
			expectedItems: []struct {
				text  string
				level int
			}{
				{"Section", 2},
				{"Subsection", 3},
			},
		},
		{
			name: "empty headings",
			input: `##

### `,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toc := extractTOC([]byte(tt.input))

			if len(toc) != tt.expectedCount {
				t.Errorf("expected %d TOC items, got %d", tt.expectedCount, len(toc))
			}

			for i, exp := range tt.expectedItems {
				if i >= len(toc) {
					break
				}
				if toc[i].Text != exp.text {
					t.Errorf("expected item %d text %q, got %q", i, exp.text, toc[i].Text)
				}
				if toc[i].Level != exp.level {
					t.Errorf("expected item %d level %d, got %d", i, exp.level, toc[i].Level)
				}
			}
		})
	}
}

func TestHighlightCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		language string
	}{
		{
			name:     "go code",
			code:     "package main\n\nfunc main() {\n    println(\"hello\")\n}",
			language: "go",
		},
		{
			name:     "python code",
			code:     "def hello():\n    print('world')",
			language: "python",
		},
		{
			name:     "javascript code",
			code:     "function hello() {\n    console.log('world');\n}",
			language: "javascript",
		},
		{
			name:     "plain text",
			code:     "some plain text",
			language: "",
		},
		{
			name:     "unknown language",
			code:     "some code",
			language: "unknownlang",
		},
		{
			name:     "empty code",
			code:     "",
			language: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := highlightCode(tt.code, tt.language)
			if err != nil {
				t.Fatalf("highlightCode failed: %v", err)
			}

			if !strings.Contains(result, "<pre") {
				t.Error("expected output to contain <pre>")
			}
		})
	}
}

func TestCodeBlockWrapper(t *testing.T) {
	wrapper := &codeBlockWrapper{}

	t.Run("start with code", func(t *testing.T) {
		result := wrapper.Start(true, ` style="color: red"`)
		if !strings.Contains(result, "<div") {
			t.Error("expected div in output")
		}
		if !strings.Contains(result, "code-block") {
			t.Error("expected code-block class")
		}
		if !strings.Contains(result, `<pre style="color: red">`) {
			t.Error("expected pre with style attribute")
		}
	})

	t.Run("start without code", func(t *testing.T) {
		result := wrapper.Start(false, "")
		if !strings.Contains(result, "<pre") {
			t.Error("expected pre in output")
		}
		if !strings.Contains(result, "code-block") {
			t.Error("expected code-block class")
		}
	})

	t.Run("end with code", func(t *testing.T) {
		result := wrapper.End(true)
		if result != "</pre></div>" {
			t.Errorf("expected '</pre></div>', got %q", result)
		}
	})

	t.Run("end without code", func(t *testing.T) {
		result := wrapper.End(false)
		if result != "</pre>" {
			t.Errorf("expected '</pre>', got %q", result)
		}
	})
}

func TestMonobrutalistChromaStyle(t *testing.T) {
	if MonobrutalistChromaStyle == nil {
		t.Error("expected MonobrutalistChromaStyle to be registered")
	}

	if MonobrutalistChromaStyle.Name == "" {
		t.Error("expected style to have a name")
	}
}

func TestMarkdownRenderer_Render_Strikethrough(t *testing.T) {
	renderer := NewMarkdownRenderer()
	input := "~~strikethrough~~"
	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<del>") {
		t.Error("expected <del> tag for strikethrough")
	}
}

func TestMarkdownRenderer_Render_TaskList(t *testing.T) {
	renderer := NewMarkdownRenderer()
	input := `- [x] Done
- [ ] Not done`
	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<ul") {
		t.Error("expected <ul> tag for list")
	}
	if !strings.Contains(result, "<li") {
		t.Error("expected <li> tag for list items")
	}
}

func TestMarkdownRenderer_Render_List(t *testing.T) {
	renderer := NewMarkdownRenderer()
	input := `- Item 1
- Item 2
  - Nested`
	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<ul") {
		t.Error("expected <ul> tag")
	}
	if !strings.Contains(result, "Item 1") {
		t.Error("expected 'Item 1' in output")
	}
	if !strings.Contains(result, "Item 2") {
		t.Error("expected 'Item 2' in output")
	}
}

func TestMarkdownRenderer_Render_CodeBlockFallback(t *testing.T) {
	renderer := NewMarkdownRenderer()
	input := "```unknown_language_that_does_not_exist\nsome code here\n```"
	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "code-block") {
		t.Error("expected code-block class in output")
	}
}

func TestTOCItem_Struct(t *testing.T) {
	item := TOCItem{
		Level: 2,
		Text:  "Test Section",
		ID:    "test-section",
	}

	if item.Level != 2 {
		t.Errorf("expected Level 2, got %d", item.Level)
	}
	if item.Text != "Test Section" {
		t.Errorf("expected Text 'Test Section', got %q", item.Text)
	}
	if item.ID != "test-section" {
		t.Errorf("expected ID 'test-section', got %q", item.ID)
	}
}
