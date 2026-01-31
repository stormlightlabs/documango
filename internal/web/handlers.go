package web

import (
	"fmt"
	"net/http"
)

// handleIndex displays the landing page with package overview.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := s.renderTemplate(w, "index.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleSearch displays search results.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "HTTP 200: Search Results (Placeholder)")
}

// handleAPISearch provides a JSON search API endpoint.
func (s *Server) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"status": "ok", "message": "Search API (Placeholder)"}`)
}

// handleDoc renders and displays a documentation page.
func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	fmt.Fprintf(w, "HTTP 200: Rendering document: %s (Placeholder)", path)
}
