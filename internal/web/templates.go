package web

import (
	"fmt"
	"html/template"
	"io"

	"github.com/stormlightlabs/documango/internal/assets"
)

var templates *template.Template

func init() {
	var err error
	templates, err = template.ParseFS(assets.TemplateFS, "templates/*.html")
	if err != nil {
		panic(fmt.Sprintf("failed to parse templates: %v", err))
	}
}

// renderTemplate executes the specified template.
func (s *Server) renderTemplate(w io.Writer, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
