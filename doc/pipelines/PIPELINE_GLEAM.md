# Gleam Ingestion Pipeline

Gleam publishes to Hex.pm and includes a machine-readable `package-interface.json` file in its documentation tarballs. This structured JSON provides direct access to documentation and type information without HTML parsing.

## Source Acquisition

Same as Elixir: `https://repo.hex.pm/docs/{package}-{version}.tar.gz`

## Data Extraction

The `package-interface.json` file is the primary data source. It contains:

- Package name, version, and Gleam version constraint
- Modules with documentation strings
- Functions with documentation, parameters (with labels and types), and return types
- Types with documentation, constructors, and type parameters
- Type aliases with their underlying types
- Constants (if any)

### Type System Rendering

Gleam's type expressions use a JSON structure with `kind` discriminators:

- `named`: Types like `List`, `Result`, `Option` with optional parameters
- `variable`: Generic type variables (rendered as `a`, `b`, `c`, etc.)
- `fn`: Function types with parameters and return type
- `tuple`: Tuple types using `elements` array (rendered as `#(a, b)`)

Function signatures are reconstructed from this JSON into Gleam syntax.

### Documentation String Format

The Gleam compiler exports documentation as either a single string or an array of strings. A custom JSON unmarshaler normalizes both formats into a concatenated string.

## Document Generation

Each module produces a comprehensive Markdown document containing:

1. Module name as heading
2. Module-level documentation
3. Types section with full definitions including constructors
4. Type aliases section with underlying type
5. Functions section with signatures and documentation

Example output:

```markdown
## Functions

### map

\`\`\`gleam
fn map(List(a), with fn(a) -> b) -> List(b)
\`\`\`

Returns a new list containing only the elements...
```

## Mapping to Unified Schema

- **Documents Table**: Each module stored as a compressed Markdown document with full type signatures
- **Search Index**: Module names, function names (prefixed with module), type names, and type aliases - all with signatures in search body
- **Agent Context**: Full Gleam function/type signatures and first-line summaries

## Not Extracted

- Implementation targets (Erlang/JavaScript compatibility flags)
- Deprecation markers
- Constants (the JSON field exists but is typically empty)

A `search-data.js` file exists with pre-rendered content for Gleam's Lunr-based search, but `package-interface.json` provides better structured data.

## Dependencies

- `repo.hex.pm` - Documentation tarballs
- `encoding/json` - Interface parsing
