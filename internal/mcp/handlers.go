package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stormlightlabs/documango/internal/codec"
	"github.com/stormlightlabs/documango/internal/db"
)

type Handlers struct {
	store *db.Store
}

func NewHandlers(store *db.Store) *Handlers {
	return &Handlers{store: store}
}

func (h *Handlers) SearchDocsHandler(ctx context.Context, req *mcp.CallToolRequest, input SearchDocsInput) (*mcp.CallToolResult, any, error) {
	results, err := h.store.SearchPackage(ctx, input.Query, input.Package, 20)
	if err != nil {
		return nil, nil, err
	}

	return nil, SearchDocsOutput{Results: results, Total: len(results)}, nil
}

func (h *Handlers) ReadDocHandler(ctx context.Context, req *mcp.CallToolRequest, input ReadDocInput) (*mcp.CallToolResult, any, error) {
	doc, err := h.store.ReadDocument(ctx, input.Path)
	if err != nil {
		return nil, nil, err
	}

	body := doc.Body
	if doc.Format == "zstd" || len(body) > 0 {
		decompressed, err := codec.Decompress(body)
		if err == nil {
			body = decompressed
		}
	}

	return nil, ReadDocOutput{Content: string(body), Format: "markdown"}, nil
}

func (h *Handlers) GetSymbolHandler(ctx context.Context, req *mcp.CallToolRequest, input GetSymbolInput) (*mcp.CallToolResult, any, error) {
	entry, err := h.store.GetSymbolContext(ctx, input.Symbol)
	if err != nil {
		return nil, nil, err
	}
	return nil, NewSymbolOutput(entry), nil
}
