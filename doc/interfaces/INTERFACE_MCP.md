# Model Context Protocol Interface

The MCP server allows LLMs to query documentation programmatically. This is the system's differentiator for AI-assisted development.

## Dependencies

Requires Go 1.23+ and the official MCP Go SDK:

```text
github.com/modelcontextprotocol/go-sdk v1.2.0+
```

Import the `mcp` and `jsonschema` packages for tool registration.

## Tool Definitions

Tools are registered using the SDK's generic `AddTool` function which automatically infers JSON schemas from Go struct types.

### search_docs

Search for documentation symbols or guides.

**Input struct**:

- `query` (string, required): Search query for documentation
- `package` (string, optional): Filter by package path prefix

**Logic**: Executes FTS5 trigram query against `search_index` using BM25 ranking. Exact matches receive a score boost.

**Output struct**: List of matches containing name, type, doc_id, and relevance score.

### read_doc

Read full content of a specific documentation page.

**Input struct**:

- `path` (string, required): Document path (e.g., "go/net/http")

**Logic**: Retrieves compressed Markdown blob from `documents` table, decompresses using zstd.

**Output**: Full Markdown content as TextContent.

### get_symbol_context

Get type signature and summary for a symbol. Minimal tokens for context efficiency.

**Input struct**:

- `symbol` (string, required): Symbol name to look up

**Logic**: Queries `agent_context` table by symbol name.

**Output struct**: Signature and summary only (not full documentation).

## Agent Workflow Example

1. User to AI: "How do I create a post in Bluesky using Go?"
2. AI calls `search_docs(query="create post", package="atproto")`
3. MCP returns list of Lexicons, e.g., `app.bsky.feed.post`
4. AI calls `read_doc(path="atproto/app.bsky.feed.post")`
5. MCP returns Markdown generated from Lexicon schema
6. AI generates correct Go code based on schema fields

## Implementation

Use official MCP Go SDK from `github.com/modelcontextprotocol/go-sdk/mcp`.

### Server Initialization

Create server with implementation metadata, register tools using `mcp.AddTool` with typed handler functions that accept input structs and return output structs. The SDK handles JSON schema inference and validation automatically.

### Transport Modes

**Stdio** (subprocess mode): Use `mcp.StdioTransport{}` with `server.Run()`. Client spawns the binary and communicates via stdin/stdout with newline-delimited JSON-RPC 2.0.

**HTTP** (network mode): Use `mcp.NewStreamableHTTPHandler()` which implements `http.Handler`. Supports bidirectional HTTP streaming.

Both transports can run simultaneously for different client needs.

### Error Handling

Tool errors should be returned as results with `IsError: true` and error text in Content, not as Go errors. Protocol-level errors propagate from `Run()`.

## Verification

Test with Claude Desktop, Claude CLI, and Antigravity to ensure:

- MCP server is discoverable via stdio transport
- Tools are callable with proper input validation
- Responses contain properly formatted TextContent
- Search results include relevance scores
- Document content is correctly decompressed
