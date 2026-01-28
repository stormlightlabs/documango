# Terminal User Interface

Built with Bubble Tea, an Elm-inspired framework ensuring composability and testability.

## Component Hierarchy

**RootModel**: Orchestrates application state, manages component lifecycle.

**SearchBubble**:

- Input field using `bubbles/textinput`
- Triggers FTS5 queries on keystroke
- Debounced for performance

**ListBubble**:

- Displays search results using `bubbles/list`
- Keyboard navigation (`j`/`k` or arrows)
- Result preview on selection

**DocBubble**:

- Content viewer using `bubbles/viewport`
- Scrollable document display
- Link detection and handling

## Markdown Rendering

Use Glamour with lipgloss for styled output based on terminal color profile.

**Custom StyleSheet**: Mimic Dash aesthetics:

- Distinct headers with color coding
- Code block backgrounds
- Syntax highlighting
- Readable typography

## Tabbed Browsing

Mirrors Dash's multi-document experience:

- `RootModel` maintains `[]DocBubble` as tab stack
- `Ctrl+Tab` to switch tabs
- Tab bar showing open documents

## Deep Linking

When user activates a link in Markdown (e.g., `[Link](std/fmt/Printf)`):

1. Intercept keypress/click event
2. Resolve path against SQLite
3. Push new view onto tab stack
4. Navigate to anchor if specified

## Key Bindings

| Key        | Action              |
|------------|---------------------|
| `/`        | Focus search        |
| `Enter`    | Open selection      |
| `Esc`      | Back/close          |
| `j`/`k`    | Navigate list       |
| `Ctrl+Tab` | Next tab            |
| `q`        | Quit                |

## Dependencies

- `charmbracelet/bubbletea` - TUI framework
- `charmbracelet/bubbles` - UI components
- `charmbracelet/glamour` - Markdown rendering
- `charmbracelet/lipgloss` - Styling
