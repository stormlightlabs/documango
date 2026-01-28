# Atproto Ingestion Pipeline

Three documentation sources for the AT Protocol and Bluesky ecosystem.

## Source 1: Lexicon Schemas

Machine-readable API definitions in `bluesky-social/atproto` under `./lexicons/`.

### Structure

Lexicon files describe NSIDs (e.g., `app.bsky.feed.post`):

- `type`: Usually `record`, `query`, or `procedure`
- `schema`: JSON-schema-like definition of fields, inputs, outputs

### Generator: Lexicon-to-Markdown

1. Parse JSON to identify "Main" definition
2. Generate H1 title from NSID
3. Render field properties (`maxLength`, `format`, `description`) into Markdown tables
4. Store raw JSON schema in `agent_context` table

### Agent Context

Raw JSON schema enables AI agents to validate their own API calls against schema constraints.

## Source 2: Protocol Specifications (atproto.com)

Hosted from `bluesky-social/atproto-website` repository. Licensed CC-BY.

### Content

- Protocol specifications (ATP, XRPC, Lexicon format)
- Architecture guides
- Federation documentation

### Ingestion

The site is built with a static generator. Two approaches:

**Markdown source**: Clone repo, process `.md` files directly from source. Minimal transformation needed.

**HTML fallback**: Fetch rendered pages, convert with html-to-markdown. Target main content selectors.

## Source 3: Developer Documentation (docs.bsky.app)

Hosted from `bluesky-social/bsky-docs` repository. Built with Docusaurus.

### Content

- Tutorials (bots, custom feeds, client apps)
- API reference (generated from OpenAPI/Lexicons)
- Developer guidelines
- Blog posts

### Ingestion

**Preferred**: Clone repo, process Docusaurus MDX source files directly.

- Handle MDX components (convert or strip)
- Preserve code examples
- Extract front matter for metadata

**Fallback**: HTML scraping with Docusaurus-specific selectors.

## Unified Namespace

All three sources indexed under `atproto/` namespace:

| Path Pattern             | Source          |
| ------------------------ | --------------- |
| `atproto/lexicon/{nsid}` | Lexicon schemas |
| `atproto/spec/{topic}`   | Protocol specs  |
| `atproto/docs/{path}`    | Developer docs  |

## Dependencies

- `github.com/bluesky-social/atproto` - Lexicon files
- `github.com/bluesky-social/atproto-website` - Protocol specs
- `github.com/bluesky-social/bsky-docs` - Developer docs
- Standard Go JSON unmarshalling
- MDX parsing (for Docusaurus source)
