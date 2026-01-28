# Go Ingestion Pipeline

Go's documentation is uniquely accessible through standardized toolchain and module proxy. We regenerate documentation from source code rather than scraping pkg.go.dev.

## Source Acquisition

**Module Proxy Protocol**: HTTP GET to `https://proxy.golang.org/<module>/@v/<version>.zip`

The tool downloads zip archives to a temporary staging area, providing raw `.go` files for analysis.

## Analysis Layers

**go/parser**: Parses source files into AST, extracts exported identifiers (names starting with uppercase).

**go/doc**: Consumes AST, computes documentation structure:

- Associates comments with functions
- Calculates method sets
- Resolves type relationships

## Transformation

Use `gomarkdoc` library to transform `go/doc` structures into GitHub-Flavored Markdown.

**Custom Renderer**:

- Input: `*doc.Package`
- Process: Iterate over Types, Funcs, Consts
- Output: Markdown stream with injected anchors for every symbol (e.g., `<a name="Client.Do"></a>`) enabling deep linking from TUI

## Agent Data Extraction

Populate `agent_context` table during Markdown generation:

**Signature Extraction**: Use `go/printer` to render function signature AST node to string.
Example: `func (c *Client) Do(req *Request) (*Response, error)`

**Summary Extraction**: Use `doc.Synopsis()` to extract first sentence of comment block.

## Dependencies

- `proxy.golang.org` - Module source
- `go/parser`, `go/doc`, `go/printer` - Standard library
- `gomarkdoc` - Markdown template logic
