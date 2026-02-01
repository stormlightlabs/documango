package web

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/stormlightlabs/documango/internal/shared"
)

// SearchResultItem represents a single search result for API responses.
type SearchResultItem struct {
	Path    string  `json:"path"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
	Package string  `json:"package"`
}

// SearchResponse represents the API search response.
type SearchResponse struct {
	Query   string             `json:"query"`
	Total   int                `json:"total"`
	Results []SearchResultItem `json:"results"`
}

// SearchErrorResponse represents an API error response.
type SearchErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func NewSearchErrorResponse(e string, c string) SearchErrorResponse {
	return SearchErrorResponse{Error: e, Code: c}
}

// SearchPageData holds data for the search template.
type SearchPageData struct {
	Query   string
	Results []SearchResultItem
	Total   int
	Package string
}

// handleAPISearch provides a JSON search API endpoint.
func (s *Server) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		s.writeSearchError(w, http.StatusBadRequest, "query parameter required", "missing_param")
		return
	}

	pkg := r.URL.Query().Get("pkg")
	limit := parseIntParam(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}
	offset := parseIntParam(r, "offset", 0)

	results, total, err := s.performSearch(ctx, query, pkg, limit, offset)
	if err != nil {
		s.writeSearchError(w, http.StatusInternalServerError, "search failed", "search_error")
		return
	}

	response := SearchResponse{
		Query:   query,
		Total:   total,
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleSearch displays search results (non-JS fallback).
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	pkg := r.URL.Query().Get("pkg")

	data := SearchPageData{
		Query:   query,
		Package: pkg,
	}

	if query != "" {
		limit := parseIntParam(r, "limit", 20)
		if limit > 100 {
			limit = 100
		}
		offset := parseIntParam(r, "offset", 0)

		results, total, err := s.performSearch(ctx, query, pkg, limit, offset)
		if err != nil {
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}

		data.Results = results
		data.Total = total
	}

	if err := s.renderTemplate(w, "search.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// performSearch executes the search query and returns results.
func (s *Server) performSearch(ctx context.Context, query, pkg string, limit, offset int) ([]SearchResultItem, int, error) {
	dbResults, err := s.store.SearchPackage(ctx, query, pkg, limit+offset)
	if err != nil {
		return nil, 0, err
	}

	if offset > len(dbResults) {
		offset = len(dbResults)
	}
	dbResults = dbResults[offset:]

	if len(dbResults) > limit {
		dbResults = dbResults[:limit]
	}

	results := make([]SearchResultItem, 0, len(dbResults))
	for _, r := range dbResults {
		doc, err := s.store.ReadDocumentByID(ctx, r.DocID)
		if err != nil {
			continue
		}

		snippet := generateSnippet(doc.Body, query)
		pkgName := extractPackageFromPath(doc.Path)

		results = append(results, SearchResultItem{
			Path:    doc.Path,
			Title:   r.Name,
			Snippet: snippet,
			Score:   r.Score,
			Package: pkgName,
		})
	}

	return results, len(results), nil
}

// writeSearchError writes a JSON error response.
func (s *Server) writeSearchError(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(NewSearchErrorResponse(message, code))
}

// parseIntParam parses an integer query parameter with a default value.
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}

// generateSnippet creates a text snippet with highlighted search terms.
func generateSnippet(body []byte, query string) string {
	if len(body) == 0 {
		return ""
	}

	text := string(body)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "  ", " ")
	text = strings.TrimSpace(text)

	terms := extractSearchTerms(query)
	if len(terms) == 0 {
		return shared.TruncateText(text, 200)
	}

	lowerText := strings.ToLower(text)
	var matchIndex int = -1
	for _, term := range terms {
		idx := strings.Index(lowerText, strings.ToLower(term))
		if idx != -1 {
			if matchIndex == -1 || idx < matchIndex {
				matchIndex = idx
			}
		}
	}

	if matchIndex == -1 {
		return shared.TruncateText(text, 200)
	}

	start := max(matchIndex-80, 0)
	end := min(matchIndex+120, len(text))

	snippet := text[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	for _, term := range terms {
		snippet = highlightTerm(snippet, term)
	}

	return snippet
}

// extractSearchTerms extracts searchable terms from the query.
func extractSearchTerms(query string) []string {
	var terms []string
	words := strings.FieldsSeq(query)

	for word := range words {
		word = strings.Trim(word, `"`)
		lower := strings.ToLower(word)

		if strings.HasPrefix(lower, "name:") ||
			strings.HasPrefix(lower, "type:") ||
			strings.HasPrefix(lower, "body:") {
			continue
		}

		if len(word) > 0 {
			terms = append(terms, word)
		}
	}

	return terms
}

// highlightTerm wraps matching terms in <mark> tags.
func highlightTerm(text, term string) string {
	if term == "" {
		return text
	}

	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(term)

	var result strings.Builder
	start := 0

	for {
		idx := strings.Index(lowerText[start:], lowerTerm)
		if idx == -1 {
			result.WriteString(text[start:])
			break
		}
		idx += start

		result.WriteString(text[start:idx])
		result.WriteString("<mark>")
		result.WriteString(text[idx : idx+len(term)])
		result.WriteString("</mark>")

		start = idx + len(term)
	}

	return result.String()
}

// extractPackageFromPath extracts the package name from a document path.
func extractPackageFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return ""
}
