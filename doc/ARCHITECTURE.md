# Architecture

## Overview

The Unified Semantic Documentation Engine (USDE) bridges human and machine documentation consumption. It provides a Dash/Zeal-like offline experience with a headless agent mode compliant with Model Context Protocol (MCP).

Go for:

- Concurrency primitives for parallel ingestion
- Bubble Tea for TUI
- SQLite FTS5 for search
- Single-file distribution

## Design Principles

**Markdown-First**: All content stored as semantic Markdown, not HTML. This serves both human readers and AI agents efficiently.

**Single-File Database**: Unlike Dash's directory bundles, USDE uses a single SQLite database (`.usde` extension) containing:

- Zstd-compressed Markdown blobs
- FTS5 search indices
- Agent-specific metadata tables

**Copyright Clean**: Custom ingestion pipelines pull from open-source repositories (Go Modules, Crates.io, Hex.pm, Bluesky GitHub) using permissively licensed content.

## Storage Engine

SQLite with FTS5 extension provides:

- Zero-configuration deployment
- Sub-millisecond query latency
- Offline capability
- Portable single-file artifacts

### Schema

**documents** - Content store replacing filesystem in legacy Dash architecture:

| Column   | Type    | Description                                              |
|----------|---------|----------------------------------------------------------|
| id       | INTEGER | Primary key                                              |
| path     | TEXT    | Virtual path (e.g., `std/net/http/Client`) for routing   |
| format   | TEXT    | Content format, usually `markdown`                       |
| body     | BLOB    | Zstd-compressed Markdown content                         |
| raw_html | BLOB    | Original HTML for fallback if conversion fails           |
| hash     | TEXT    | SHA256 checksum for change detection during updates      |

**search_index** - FTS5 virtual table for natural language querying:

| Column | Description                                  |
|--------|----------------------------------------------|
| name   | Symbol name (e.g., `GenServer`, `Option`)    |
| type   | Semantic type (e.g., `Module`, `Trait`)      |
| body   | Full text content for deep search            |
| doc_id | Foreign key to documents                     |

**agent_context** - Low-token-count data for AI agents:

| Column    | Type    | Description                                                 |
|-----------|---------|-----------------------------------------------------------  |
| doc_id    | INTEGER | Foreign key                                                 |
| symbol    | TEXT    | Exact symbol identifier                                     |
| signature | TEXT    | Type signature (e.g., `fn map<U>(self, f: F) -> Option<U>`) |
| summary   | TEXT    | First paragraph of docstring                                |

### Search Implementation

**Trigram Tokenization**: FTS5 configured with trigram tokenizer for substring matching and fuzzy search.
`Println` becomes `Pri`, `rin`, `int`, `ntl`, `tln`.

**Custom Ranking**:

```sql
SELECT *,
  (CASE WHEN name = $query THEN 100 ELSE 0 END) + bm25(search_index) AS score
FROM search_index
ORDER BY score DESC;
```

Exact matches rank above partial matches (e.g., `Client` before `NewClient`).

## Architectural Comparison

### Dash/Zeal Limitations

- **Inode Exhaustion**: Loose HTML files cause massive inode usage
- **HTML Opacity**: Agents must parse HTML, strip tags, interpret CSS classes
- **Copyright Concerns**: Often relies on scraping proprietary sites

### DevDocs Limitations

- Stores in browser IndexedDB as compressed JSON
- Relies on DOM for rendering
- Unsuitable for terminal environments

### USDE Advantages

- Single portable file
- Semantic Markdown substrate
- Dual-mode: human TUI + agent MCP
- Direct source ingestion
