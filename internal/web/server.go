package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stormlightlabs/documango/internal/assets"
	"github.com/stormlightlabs/documango/internal/db"
)

// Server represents the web documentation server.
type Server struct {
	store  *db.Store
	router *http.ServeMux
	addr   string
}

// NewServer creates a new instance of the web server.
func NewServer(store *db.Store, addr string) *Server {
	s := &Server{
		store:  store,
		router: http.NewServeMux(),
		addr:   addr,
	}
	s.registerRoutes()
	return s
}

// registerRoutes sets up the HTTP router with placeholder handlers.
func (s *Server) registerRoutes() {
	s.router.HandleFunc("GET /", s.handleIndex)
	s.router.HandleFunc("GET /search", s.handleSearch)
	s.router.HandleFunc("GET /api/search", s.handleAPISearch)
	s.router.HandleFunc("GET /doc/{path...}", s.handleDoc)
	s.router.Handle("GET /static/", http.FileServer(http.FS(assets.StaticFS)))
}

// Start runs the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:    s.addr,
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	fmt.Printf("Web interface listening on http://%s\n", s.addr)
	return server.ListenAndServe()
}
