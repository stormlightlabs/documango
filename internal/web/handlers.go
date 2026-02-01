package web

import (
	"context"
	"net/http"
	"path"
	"strings"

	"github.com/stormlightlabs/documango/internal/db"
)

// PackageGroup represents a group of packages by language.
type PackageGroup struct {
	Language string
	Packages []db.PackageInfo
}

// IndexPageData holds data for the index template.
type IndexPageData struct {
	Groups []PackageGroup
}

// handleIndex displays the landing page with package overview.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	packages, err := s.store.ListPackages(ctx)
	if err != nil {
		http.Error(w, "Failed to load packages", http.StatusInternalServerError)
		return
	}

	groupMap := make(map[string][]db.PackageInfo)
	for _, pkg := range packages {
		groupMap[pkg.Language] = append(groupMap[pkg.Language], pkg)
	}

	var groups []PackageGroup
	for lang, pkgs := range groupMap {
		groups = append(groups, PackageGroup{
			Language: lang,
			Packages: pkgs,
		})
	}

	data := IndexPageData{Groups: groups}
	if err := s.renderTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
