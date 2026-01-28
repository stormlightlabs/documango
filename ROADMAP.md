# Roadmap

## [Done] Milestone 1: Foundation

Core CLI and database infrastructure.

- Define SQLite schema (documents, search_index, agent_context tables)
- Implement FTS5 wrapper using `modernc.org/sqlite` (pure Go, no CGO)
- Build CLI shell with fang (cobra) & lipgloss
- Zstd compression/decompression for document blobs
- Database creation, opening, and basic CRUD operations

**DoD**: Can create `.usde` database, insert documents, and run FTS5 queries.

## [Done] Milestone 2: Go Ingestion

First complete ingestion pipeline.

- Module proxy client for downloading source archives
- AST parsing with `go/parser` and `go/doc`
- Markdown generation using gomarkdoc templates
- Signature extraction with `go/printer`
- Summary extraction with `doc.Synopsis()`
- Anchor injection for deep linking
- Populate all three tables from Go module source

**DoD**: Can ingest a Go module (e.g., `net/http`) and produce searchable documentation.

## [Done] Milestone 3: Atproto Ingestion

Three documentation sources for AT Protocol ecosystem.

- Clone/fetch atproto, atproto-website, and bsky-docs repositories
- JSON Lexicon parser with NSID-to-Markdown generator
- Markdown source processing for protocol specs (atproto-website)
- MDX/Docusaurus processing for developer docs (bsky-docs)
- Unified namespace: `atproto/lexicon/`, `atproto/spec/`, `atproto/docs/`
- Raw schema storage in agent_context for Lexicons

**Definition of Done (DoD)**: Can ingest Lexicons, protocol specs, and developer tutorials from all three sources.

## [Done] Milestone 4: Storage Layer & CLI Redesign

Persistent caching and improved command interface.

- XDG Base Directory compliance with `DOCUMANGO_HOME` override
- Cache layer for remote sources (module zips, stdlib tarballs, git clones)
- Cache manifest with ETags, timestamps, and checksums for invalidation
- Fetch-through pattern wrapping existing network operations
- CLI restructure: `init`, `add`, `search`, `read`, `list`, `info`, `cache`, `config`
- POSIX-compliant flags with consistent global options (`-d`, `-v`, `-q`)
- Output format options: table, JSON, paths
- Configuration file support (TOML) for defaults and preferences
- Standard exit codes and stream conventions

**DoD**: Remote sources cached locally, new CLI commands functional.

## [Done] Milestone 5: Hex.pm Ingestion (Elixir + Gleam)

HTML-based ingestion for Hex packages.

- Hex.pm tarball downloader (shared)
- Goquery HTML parsing
- html-to-markdown conversion with ExDoc plugins
- Gleam doc generator selector mappings
- Admonition block handling
- sidebar_items.json parsing for arity
- Typespec extraction

**DoD**: Can ingest Elixir (e.g., Phoenix) and Gleam packages from Hex.pm.

- Test with:
    1. <https://hexdocs.pm/gleam_stdlib/>
    2. <https://hexdocs.pm/phoenix/>
    3. <https://hexdocs.pm/elixir/>

## Milestone 6: Rust Ingestion

Dual-path ingestion strategy.

- Rustdoc JSON parser (nightly path)
- JSON-to-Markdown transformation
- Re-export resolution
- Docs.rs HTML fallback scraper
- Path selection logic based on toolchain availability

**DoD**: Can ingest crates via either path and produce searchable documentation.

- Test with:
    1. <https://docs.rs/pulldown-cmark/>
    2. <https://docs.rs/tokio/>
    3. <https://docs.rs/axum/>

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

- MCP server implementation using official Go SDK
- `search_docs` tool with trigram search
- `read_doc` tool with decompression
- `get_symbol_context` tool for minimal token responses
- Stdio and HTTP transport modes
- Verify with Claude Desktop

**DoD**: AI agent can search and retrieve documentation programmatically.

## Milestone 11: Polish

Production readiness.

- Incremental updates using SHA256 checksums
- Error handling and recovery
- Cross-platform testing
- Documentation and examples

**DoD**: Reliable, deployable tool ready for daily use.
