package mcp

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stormlightlabs/documango/internal/db"
)

// NewServer creates a new MCP server for documango.
func NewServer(store *db.Store, version string) *mcp.Server {
	logger := slog.New(slog.NewJSONHandler(
		os.Stderr,
		&slog.HandlerOptions{Level: slog.LevelInfo},
	))

	server := mcp.NewServer(
		&mcp.Implementation{Name: "documango", Version: version},
		&mcp.ServerOptions{Logger: logger},
	)

	handlers := NewHandlers(store)

	mcp.AddTool(server, newTool("search_docs", "Search for documentation symbols or guides"),
		func(ctx context.Context, req *mcp.CallToolRequest, input SearchDocsInput) (*mcp.CallToolResult, any, error) {
			logger.Info("Tool call: search_docs", "query", input.Query, "package", input.Package)
			return handlers.SearchDocsHandler(ctx, req, input)
		})

	mcp.AddTool(server, newTool("read_doc", "Read full content of a specific documentation page"),
		func(ctx context.Context, req *mcp.CallToolRequest, input ReadDocInput) (*mcp.CallToolResult, any, error) {
			logger.Info("Tool call: read_doc", "path", input.Path)
			return handlers.ReadDocHandler(ctx, req, input)
		})

	mcp.AddTool(server, newTool("get_symbol_context", "Get type signature and summary for a symbol"),
		func(ctx context.Context, req *mcp.CallToolRequest, input GetSymbolInput) (*mcp.CallToolResult, any, error) {
			logger.Info("Tool call: get_symbol_context", "symbol", input.Symbol)
			return handlers.GetSymbolHandler(ctx, req, input)
		})

	return server
}

// RunStdio runs the server using the stdio transport.
func RunStdio(ctx context.Context, server *mcp.Server) error {
	return server.Run(ctx, &mcp.StdioTransport{})
}

// RunHTTP runs the server using the streamable HTTP transport.
func RunHTTP(ctx context.Context, server *mcp.Server, addr string) error {
	f := func(r *http.Request) *mcp.Server { return server }
	handler := mcp.NewStreamableHTTPHandler(f, nil)

	s := &http.Server{Addr: addr, Handler: handler}

	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()

	return s.ListenAndServe()
}

func newTool(n, d string) *mcp.Tool {
	return &mcp.Tool{Name: n, Description: d}
}
