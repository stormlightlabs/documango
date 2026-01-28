package mcp

import "github.com/stormlightlabs/documango/internal/db"

// SearchDocsInput defines the input schema for the search_docs tool.
type SearchDocsInput struct {
	Query   string `json:"query" jsonschema:"Search query for documentation"`
	Package string `json:"package,omitempty" jsonschema:"Filter by package path prefix"`
}

// SearchDocsOutput defines the output schema for the search_docs tool.
type SearchDocsOutput struct {
	Results []db.SearchResult `json:"results"`
	Total   int               `json:"total"`
}

// ReadDocInput defines the input schema for the read_doc tool.
type ReadDocInput struct {
	Path string `json:"path" jsonschema:"Document path (e.g., 'go/net/http')"`
}

// ReadDocOutput defines the output schema for the read_doc tool.
type ReadDocOutput struct {
	Content string `json:"content"`
	Format  string `json:"format"`
}

// GetSymbolInput defines the input schema for the get_symbol_context tool.
type GetSymbolInput struct {
	Symbol string `json:"symbol" jsonschema:"Symbol name to look up"`
}

// GetSymbolOutput defines the output schema for the get_symbol_context tool.
type GetSymbolOutput struct {
	Symbol    string `json:"symbol"`
	Signature string `json:"signature"`
	Summary   string `json:"summary"`
}

func NewSymbolOutput(entry db.AgentContext) GetSymbolOutput {
	return GetSymbolOutput{Symbol: entry.Symbol, Signature: entry.Signature, Summary: entry.Summary}
}
