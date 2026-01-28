# Web Interface

A `serve` command provides browser-based documentation reading for users who prefer graphical interfaces.

## HTTP Server

Built on Go's `net/http` standard library.

## Dynamic Rendering

Request flow for `/doc/fmt/Printf`:

1. Parse path, extract document identifier
2. Query SQLite for Markdown content
3. Decompress Zstd blob
4. Render Markdown to HTML using Goldmark
5. Inject CSS stylesheet
6. Return complete HTML page

## Styling

Use GitHub Markdown CSS or similar for:

- Readable typography
- Code block highlighting
- Table formatting
- Responsive layout

## TUI Synchronization

Websocket connection between browser and TUI using `gorilla/websocket`:

**Leader/Follower Mode**:

- Selection in TUI auto-scrolls browser view
- Enables side-by-side terminal + browser workflow
- Useful for presentations or pair programming

## Endpoints

| Path           | Description                    |
|----------------|--------------------------------|
| `/`            | Search interface               |
| `/doc/{path}`  | Render document                |
| `/api/search`  | JSON search results            |
| `/ws`          | Websocket for sync             |

## Dependencies

- `net/http` - Standard library
- `yuin/goldmark` - Markdown to HTML
- `gorilla/websocket` - Real-time sync
