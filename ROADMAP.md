# Roadmap

See [CHANGELOG](CHANGELOG.md) for completed work.

## Milestone 7: GitHub Markdown

Repository documentation ingestion.

- GitHub API client with rate limiting
- Raw content fetching for public repos
- Repository tree traversal for markdown discovery
    - Search repos for doc/, docs/, README, LICENSE, CHANGELOG, etc
- Relative link resolution
- Front matter extraction
- Clone-based fallback for large repos

**DoD**: Can ingest markdown documentation from GitHub repositories.

## Milestone 8: TUI

Interactive terminal interface.

- Bubble Tea application structure with RootModel
- SearchBubble with keystroke-triggered FTS5 queries
- ListBubble for result navigation
- DocBubble with Glamour markdown rendering
- Custom stylesheet matching Dash aesthetics
- Tab stack for multi-document browsing
- Deep link handling within documents

**DoD**: Full read-only documentation browsing experience in terminal.

## Milestone 9: Web Interface

Browser-based reading.

- HTTP server with document routing
- Goldmark Markdown-to-HTML rendering
    - CSS stylesheet injection
- Search API endpoint
- Websocket TUI synchronization

**DoD**: Can browse documentation in web browser with TUI sync.

## Milestone 10: MCP Server

AI agent integration.

- MCP server implementation using official [Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- `search_docs` tool with trigram search
- `read_doc` tool with decompression
- `get_symbol_context` tool for minimal token responses
- Stdio and HTTP transport modes
- Verify with Claude Desktop, Claude CLI, & Antigravity

**DoD**: AI agent can search and retrieve documentation programmatically.

## Milestone 11: Polish

Production readiness.

- Incremental updates using SHA256 checksums
- Error handling and recovery
- Cross-platform testing
- Documentation and examples
- Goreleaser integration/action

**DoD**: Reliable, deployable tool ready for daily use.
