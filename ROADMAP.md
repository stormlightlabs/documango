# Roadmap

See [CHANGELOG](CHANGELOG.md) for completed work.

## Milestone 8: MCP Server

AI agent integration. No dependencies on TUI or Web interfaces.

- MCP server implementation using official [Go SDK](https://github.com/modelcontextprotocol/go-sdk) v1.2.0+
- `search_docs` tool with FTS5 trigram search and BM25 ranking
- `read_doc` tool with zstd decompression
- `get_symbol_context` tool for minimal token responses
- Stdio transport (subprocess mode for Claude Desktop/CLI)
- HTTP transport (network mode via StreamableHTTPHandler)
- CLI subcommand: `documango mcp serve`
- Verify with Claude Desktop, Claude CLI, & Antigravity

**DoD**: AI agent can search and retrieve documentation programmatically.

## Milestone 9: Web Interface

Browser-based reading.

- HTTP server with document routing
- Goldmark Markdown-to-HTML rendering with CSS injection
- Search API endpoint with JSON responses
- Dark terminal aesthetic: Geist for headings, Geist Mono for body
- Readability-focused layout

**DoD**: Can browse and search documentation in web browser.

## Milestone 10: TUI

Interactive terminal interface.

- Bubble Tea application structure with RootModel
- SearchBubble with keystroke-triggered FTS5 queries
- ListBubble for result navigation
- DocBubble with Glamour markdown rendering
- Custom stylesheet matching Dash aesthetics
- Tab stack for multi-document browsing
- Deep link handling within documents

**DoD**: Full read-only documentation browsing experience in terminal.

## Milestone 11: Endgame

Production readiness.

- Incremental updates using SHA256 checksums
- Error handling and recovery
- Cross-platform testing
- Documentation and examples
- Goreleaser integration/action
- TUI-Web synchronization via Websocket

**DoD**: Reliable, deployable tool ready for daily use.
