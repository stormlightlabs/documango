# Model Context Protocol Interface

The MCP server allows LLMs to query documentation programmatically. This is the system's differentiator for AI-assisted development.

## Tool Definitions

### search_docs

Search for documentation symbols or guides.

**Input Schema**:

```json
{
  "query": "string",
  "language": "string (optional)"
}
```

**Logic**: Executes FTS5 trigram query against `search_index`.

**Output**: JSON list of matches with summaries.

### read_doc

Read full content of a specific documentation page.

**Input Schema**:

```json
{
  "path": "string"
}
```

**Logic**: Retrieves Markdown blob from `documents` table, decompresses.

**Output**: Full Markdown content.

### get_symbol_context

Get type signature and summary for a symbol. Minimal tokens for context efficiency.

**Input Schema**:

```json
{
  "symbol": "string"
}
```

**Logic**: Queries `agent_context` table.

**Output**: Signature and summary only (not full documentation).

## Agent Workflow Example

1. User to AI: "How do I create a post in Bluesky using Go?"
2. AI calls `search_docs(query="create post", language="atproto")`
3. MCP returns list of Lexicons, e.g., `app.bsky.feed.post`
4. AI calls `read_doc(path="app.bsky.feed.post")`
5. MCP returns Markdown generated from Lexicon schema
6. AI generates correct Go code based on schema fields

## Implementation

Use official MCP Go SDK from `github.com/modelcontextprotocol/go-sdk/mcp`.

Server runs as:

- Subprocess spawned by AI client
- Standalone daemon with stdio transport
- HTTP server for network access

## Verification

Test with Claude Desktop to ensure:

- MCP server is discoverable
- Tools are callable
- Responses are properly formatted
