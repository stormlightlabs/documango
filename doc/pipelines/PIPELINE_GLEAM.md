# Gleam Ingestion Pipeline

Gleam publishes to Hex.pm but uses its own documentation generator, not ExDoc. The HTML structure differs from Elixir packages.

## Source Acquisition

Same as Elixir: `https://repo.hex.pm/docs/{package}-{version}.tar.gz`

## HTML Structure Differences

Gleam's doc generator:

- Uses highlight.js with custom Gleam language definitions
- Different CSS class conventions than ExDoc
- Generates search index separately
- Includes static assets (CSS, JS, fonts)

## Parsing Strategy

Use Goquery with Gleam-specific selectors:

- Module documentation structure
- Function/type/constant definitions
- Code blocks with Gleam syntax highlighting classes

## Conversion

Strip highlight.js classes, produce standard fenced blocks:

```gleam
pub fn example() -> String
```

## Agent Context

Gleam's type system is similar to ML languages. Capture:

- Function signatures with types
- Type definitions
- Module hierarchy

## Shared Infrastructure with Elixir

- Same Hex.pm tarball download logic
- Same html-to-markdown core library
- Different selector mappings and post-processing

## Dependencies

- `repo.hex.pm` - Documentation tarballs
- `goquery` - HTML parsing
- `html-to-markdown` - Conversion library
