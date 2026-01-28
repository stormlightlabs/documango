# CHANGELOG

## [0.1.0] - 2026-01-28

### Added

- Core application infrastructure with a SQLite database schema, FTS5 search wrapper, and CLI shell using Fang (Cobra) and Lipgloss.

- Go module & package ingestion pipeline with AST parsing, signature extraction, and automatic markdown generation.

- AT Protocol ecosystem ingestion of Lexicons, protocol specifications, and developer documentation from multiple sources.

- XDG-compliant persistent caching layer for managing remote sources.

- Elixir and Gleam package doc ingestion from Hex.pm, utilizing HTML parsing and html-to-markdown conversion.

- Dual-path ingestion strategy for Rust crates, supporting both Rustdoc JSON parsing and a Docs.rs HTML fallback scraper.
