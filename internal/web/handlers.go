package web

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/stormlightlabs/documango/internal/db"
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

// DocPageData holds the data for the document template.
type DocPageData struct {
	Title       string
	Path        string
	Breadcrumbs []BreadcrumbItem
	Content     string
	TOC         []TOCItem
}

// BreadcrumbItem represents a single breadcrumb entry.
type BreadcrumbItem struct {
	Label string
	URL   string
}

// handleDoc renders and displays a documentation page.
func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	docPath := r.PathValue("path")

	if docPath == "" {
		http.Error(w, "Document path is required", http.StatusBadRequest)
		return
	}

	doc, err := s.fetchDocument(ctx, docPath)
	if err != nil {
		s.handleDocError(w, r, err)
		return
	}

	renderer := NewMarkdownRenderer()
	htmlContent, toc, err := renderer.RenderWithTOC(doc.Body)
	if err != nil {
		http.Error(w, "Failed to render document", http.StatusInternalServerError)
		return
	}

	data := DocPageData{
		Title:       extractTitle(doc, docPath),
		Path:        docPath,
		Breadcrumbs: buildBreadcrumbs(docPath),
		Content:     htmlContent,
		TOC:         toc,
	}

	if err := s.renderTemplate(w, "doc.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// fetchDocument retrieves a document from the store by path.
func (s *Server) fetchDocument(ctx context.Context, docPath string) (db.Document, error) {
	doc, err := s.store.ReadDocument(ctx, docPath)
	if err != nil {
		return db.Document{}, err
	}
	return doc, nil
}

// handleDocError handles errors when fetching or rendering documents.
func (s *Server) handleDocError(w http.ResponseWriter, _ *http.Request, err error) {
	if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no rows") {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}
	http.Error(w, "Failed to load document", http.StatusInternalServerError)
}

// extractTitle extracts a title from the document or uses the path.
func extractTitle(doc db.Document, docPath string) string {
	if doc.Format == "markdown" || doc.Format == "md" {
		body := string(doc.Body)
		if strings.HasPrefix(body, "# ") {
			end := strings.Index(body, "\n")
			if end == -1 {
				end = len(body)
			}
			return strings.TrimSpace(body[2:end])
		}
	}
	return path.Base(docPath)
}

// buildBreadcrumbs creates breadcrumb items from the document path.
func buildBreadcrumbs(docPath string) []BreadcrumbItem {
	parts := strings.Split(strings.Trim(docPath, "/"), "/")
	items := make([]BreadcrumbItem, 0, len(parts)+1)

	items = append(items, BreadcrumbItem{Label: "Home", URL: "/"})

	var currentPath strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		currentPath.WriteString("/")
		currentPath.WriteString(part)

		item := BreadcrumbItem{Label: part, URL: "/doc" + currentPath.String()}
		if i == len(parts)-1 {
			item.URL = ""
		}
		items = append(items, item)
	}

	return items
}
