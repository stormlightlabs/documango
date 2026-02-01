# Roadmap

See [CHANGELOG](CHANGELOG.md) for completed work.

## [DONE] Milestone 9: Web Interface

Browser-based documentation reader with Monobrutalist design system.

- HTTP server with document routing (`/doc/{path}`, `/search`, `/api/search`)
- Goldmark Markdown-to-HTML rendering with custom Chroma theme
- Search API endpoint with JSON responses and snippet highlighting
- Monobrutalist CSS: Geist/Geist Mono typography, 2px borders, hard shadows, dark terminal palette
- Embedded templates and static assets via `//go:embed`

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
