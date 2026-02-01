package web

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// MonobrutalistChromaStyle is a custom Chroma theme matching the Monobrutalist design system.
var MonobrutalistChromaStyle = styles.Register(chroma.MustNewStyle("monobrutalist", chroma.StyleEntries{
	chroma.Text:                 "#fafafa",
	chroma.Error:                "#ef4444",
	chroma.Comment:              "#525252",
	chroma.CommentPreproc:       "#a1a1aa",
	chroma.Keyword:              "#22c55e",
	chroma.KeywordReserved:      "#22c55e",
	chroma.KeywordNamespace:     "#22c55e",
	chroma.KeywordType:          "#22c55e",
	chroma.Operator:             "#fafafa",
	chroma.Punctuation:          "#a1a1aa",
	chroma.Name:                 "#fafafa",
	chroma.NameBuiltin:          "#22c55e",
	chroma.NameTag:              "#22c55e",
	chroma.NameAttribute:        "#a1a1aa",
	chroma.NameClass:            "#22c55e",
	chroma.NameConstant:         "#22c55e",
	chroma.NameDecorator:        "#22c55e",
	chroma.NameException:        "#ef4444",
	chroma.NameFunction:         "#22c55e",
	chroma.NameProperty:         "#a1a1aa",
	chroma.NameLabel:            "#22c55e",
	chroma.NameNamespace:        "#fafafa",
	chroma.NameOther:            "#fafafa",
	chroma.NameVariable:         "#fafafa",
	chroma.NameVariableMagic:    "#22c55e",
	chroma.Literal:              "#fafafa",
	chroma.LiteralDate:          "#a1a1aa",
	chroma.LiteralString:        "#a3e635",
	chroma.LiteralStringAffix:   "#22c55e",
	chroma.LiteralStringEscape:  "#22c55e",
	chroma.LiteralStringRegex:   "#a3e635",
	chroma.LiteralNumber:        "#f97316",
	chroma.LiteralNumberBin:     "#f97316",
	chroma.LiteralNumberFloat:   "#f97316",
	chroma.LiteralNumberHex:     "#f97316",
	chroma.LiteralNumberInteger: "#f97316",
	chroma.LiteralNumberOct:     "#f97316",
	chroma.Generic:              "#fafafa",
	chroma.GenericDeleted:       "#ef4444",
	chroma.GenericEmph:          "italic",
	chroma.GenericError:         "#ef4444",
	chroma.GenericHeading:       "#fafafa bold",
	chroma.GenericInserted:      "#22c55e",
	chroma.GenericOutput:        "#a1a1aa",
	chroma.GenericPrompt:        "#525252",
	chroma.GenericStrong:        "bold",
	chroma.GenericSubheading:    "#fafafa bold",
	chroma.GenericTraceback:     "#ef4444",
	chroma.Background:           "#1f1f1f",
}))

// MarkdownRenderer handles markdown to HTML conversion with syntax highlighting.
type MarkdownRenderer struct {
	md goldmark.Markdown
}

// NewMarkdownRenderer creates a new markdown renderer with all extensions enabled.
func NewMarkdownRenderer() *MarkdownRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			&headingAnchorExt{},
			&codeHighlightExt{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			renderer.WithNodeRenderers(
				util.Prioritized(&customHTMLRenderer{}, 100),
			),
		),
	)

	return &MarkdownRenderer{md: md}
}

// Render converts markdown content to HTML with syntax highlighting.
func (r *MarkdownRenderer) Render(source []byte) (string, error) {
	var buf bytes.Buffer
	if err := r.md.Convert(source, &buf); err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}
	return buf.String(), nil
}

// RenderWithTOC converts markdown to HTML and extracts table of contents.
func (r *MarkdownRenderer) RenderWithTOC(source []byte) (html string, toc []TOCItem, err error) {
	toc = extractTOC(source)
	html, err = r.Render(source)
	if err != nil {
		return "", nil, err
	}

	return html, toc, nil
}

// TOCItem represents a single entry in the table of contents.
type TOCItem struct {
	Level int
	Text  string
	ID    string
}

// extractTOC parses markdown and extracts heading structure for table of contents.
func extractTOC(source []byte) []TOCItem {
	var toc []TOCItem

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)

	doc := md.Parser().Parse(text.NewReader(source))

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		if heading.Level < 2 || heading.Level > 3 {
			return ast.WalkContinue, nil
		}

		var text bytes.Buffer
		for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				text.Write(textNode.Value(source))
			}
		}

		id, _ := heading.AttributeString("id")
		idStr := ""
		if id != nil {
			idStr = string(id.([]byte))
		}

		toc = append(toc, TOCItem{
			Level: heading.Level,
			Text:  strings.TrimSpace(text.String()),
			ID:    idStr,
		})

		return ast.WalkContinue, nil
	})

	return toc
}

// headingAnchorExt adds anchor links to headings.
type headingAnchorExt struct{}

func (e *headingAnchorExt) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&headingAnchorRenderer{}, 100),
	))
}

type headingAnchorRenderer struct{}

func (r *headingAnchorRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.renderHeading)
}

func (r *headingAnchorRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		id, _ := n.AttributeString("id")
		idStr := ""
		if id != nil {
			idStr = string(id.([]byte))
		}

		_, _ = w.WriteString("<h")
		_ = w.WriteByte("0123456"[n.Level])
		if idStr != "" {
			_, _ = w.WriteString(` id="`)
			_, _ = w.WriteString(idStr)
			_, _ = w.WriteString(`"`)
		}
		_, _ = w.WriteString(">")

		if idStr != "" {
			_, _ = w.WriteString(`<a href="#`)
			_, _ = w.WriteString(idStr)
			_, _ = w.WriteString(`" class="anchor" aria-hidden="true">#</a>`)
		}
	} else {
		_, _ = w.WriteString("</h")
		_ = w.WriteByte("0123456"[n.Level])
		_, _ = w.WriteString(">\n")
	}
	return ast.WalkContinue, nil
}

// codeHighlightExt provides syntax highlighting for code blocks.
type codeHighlightExt struct{}

func (e *codeHighlightExt) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&codeBlockRenderer{}, 100),
	))
}

type codeBlockRenderer struct{}

func (r *codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderCodeBlock)
}

func (r *codeBlockRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*ast.FencedCodeBlock)

	language := string(n.Language(source))
	if language == "" {
		language = "text"
	}

	var code bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(source))
	}

	highlighted, err := highlightCode(code.String(), language)
	if err != nil {
		_, _ = w.WriteString("<pre class=\"code-block\"><code>")
		_, _ = w.WriteString(code.String())
		_, _ = w.WriteString("</code></pre>\n")
		return ast.WalkContinue, nil
	}

	_, _ = w.WriteString(highlighted)
	return ast.WalkSkipChildren, nil
}

// highlightCode applies syntax highlighting using Chroma.
func highlightCode(code, language string) (string, error) {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := MonobrutalistChromaStyle
	formatter := chromahtml.New(
		chromahtml.WithLineNumbers(false),
		chromahtml.WithPreWrapper(&codeBlockWrapper{}),
	)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// codeBlockWrapper wraps code blocks with custom HTML structure.
type codeBlockWrapper struct{}

func (w *codeBlockWrapper) Start(code bool, styleAttr string) string {
	if code {
		return fmt.Sprintf(`<div class="code-block"><pre%s>`, styleAttr)
	}
	return `<pre class="code-block">`
}

func (w *codeBlockWrapper) End(code bool) string {
	if code {
		return "</pre></div>"
	}
	return "</pre>"
}

// customHTMLRenderer adds CSS classes to rendered elements.
type customHTMLRenderer struct{}

func (r *customHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extast.KindTable, r.renderTable)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
}

func (r *customHTMLRenderer) renderTable(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<table class="doc-table">`)
	} else {
		_, _ = w.WriteString("</table>\n")
	}
	return ast.WalkContinue, nil
}

func (r *customHTMLRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<blockquote class="callout">`)
	} else {
		_, _ = w.WriteString("</blockquote>\n")
	}
	return ast.WalkContinue, nil
}

func (r *customHTMLRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<code class="code-inline">`)
	} else {
		_, _ = w.WriteString("</code>")
	}
	return ast.WalkContinue, nil
}
