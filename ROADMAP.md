# Roadmap

## Milestone 1: Foundation

Core CLI and database infrastructure.

- Define SQLite schema (documents, search_index, agent_context tables)
- Implement FTS5 wrapper using `modernc.org/sqlite` (pure Go, no CGO)
- Build CLI shell with cobra
- Zstd compression/decompression for document blobs
- Database creation, opening, and basic CRUD operations

**Exit Criteria**: Can create `.usde` database, insert documents, and run FTS5 queries.

## Milestone 2: Go Ingestion

First complete ingestion pipeline.

- Module proxy client for downloading source archives
- AST parsing with `go/parser` and `go/doc`
- Markdown generation using gomarkdoc templates
- Signature extraction with `go/printer`
- Summary extraction with `doc.Synopsis()`
- Anchor injection for deep linking
- Populate all three tables from Go module source

**Exit Criteria**: Can ingest a Go module (e.g., `net/http`) and produce searchable documentation.

## Milestone 3: Atproto Ingestion

Three documentation sources for AT Protocol ecosystem.

- Clone/fetch atproto, atproto-website, and bsky-docs repositories
- JSON Lexicon parser with NSID-to-Markdown generator
- Markdown source processing for protocol specs (atproto-website)
- MDX/Docusaurus processing for developer docs (bsky-docs)
- Unified namespace: `atproto/lexicon/`, `atproto/spec/`, `atproto/docs/`
- Raw schema storage in agent_context for Lexicons

**Exit Criteria**: Can ingest Lexicons, protocol specs, and developer tutorials from all three sources.

## Milestone 4: TUI

Interactive terminal interface.

- Bubble Tea application structure with RootModel
- SearchBubble with keystroke-triggered FTS5 queries
- ListBubble for result navigation
- DocBubble with Glamour markdown rendering
- Custom stylesheet matching Dash aesthetics
- Tab stack for multi-document browsing
- Deep link handling within documents

**Exit Criteria**: Full read-only documentation browsing experience in terminal.

## Milestone 5: Hex.pm Ingestion (Elixir + Gleam)

HTML-based ingestion for Hex packages.

- Hex.pm tarball downloader (shared)
- Goquery HTML parsing
- html-to-markdown conversion with ExDoc plugins
- Gleam doc generator selector mappings
- Admonition block handling
- sidebar_items.json parsing for arity
- Typespec extraction

**Exit Criteria**: Can ingest Elixir (e.g., Phoenix) and Gleam packages from Hex.pm.

## Milestone 6: Rust Ingestion

Dual-path ingestion strategy.

- Rustdoc JSON parser (nightly path)
- JSON-to-Markdown transformation
- Re-export resolution
- Docs.rs HTML fallback scraper
- Path selection logic based on toolchain availability

**Exit Criteria**: Can ingest crates via either path and produce searchable documentation.

## Milestone 7: GitHub Markdown

Repository documentation ingestion.

- GitHub API client with rate limiting
- Raw content fetching for public repos
- Repository tree traversal for markdown discovery
- Relative link resolution
- Front matter extraction
- Clone-based fallback for large repos

**Exit Criteria**: Can ingest markdown documentation from GitHub repositories.

## Milestone 8: Web Interface

Browser-based reading.

- HTTP server with document routing
- Goldmark Markdown-to-HTML rendering
- CSS stylesheet injection
- Search API endpoint
- Websocket TUI synchronization

**Exit Criteria**: Can browse documentation in web browser with TUI sync.

## Milestone 9: MCP Server

AI agent integration.

- MCP server implementation using official Go SDK
- `search_docs` tool with trigram search
- `read_doc` tool with decompression
- `get_symbol_context` tool for minimal token responses
- Stdio and HTTP transport modes
- Verify with Claude Desktop

**Exit Criteria**: AI agent can search and retrieve documentation programmatically.

## Milestone 10: Polish

Production readiness.

- Caching layer for slow network operations
- Incremental updates using SHA256 checksums
- Error handling and recovery
- Cross-platform testing
- Documentation and examples

**Exit Criteria**: Reliable, deployable tool ready for daily use.
