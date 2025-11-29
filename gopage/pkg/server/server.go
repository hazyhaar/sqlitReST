package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/horos/gopage/pkg/engine"
	"github.com/horos/gopage/pkg/render"
)

// Config holds server configuration
type Config struct {
	Port        int
	DBPath      string
	SQLDir      string
	TemplateDir string
	AssetsDir   string
	Debug       bool
}

// Server represents the GoPage HTTP server
type Server struct {
	config   Config
	router   *chi.Mux
	executor *engine.SQLExecutor
	renderer *render.Renderer
	parser   *engine.SQLParser
}

// New creates a new GoPage server
func New(cfg Config) (*Server, error) {
	// Create SQL executor
	executor, err := engine.NewSQLExecutor(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL executor: %w", err)
	}

	// Create renderer
	renderer, err := render.NewRenderer(cfg.TemplateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	s := &Server{
		config:   cfg,
		router:   chi.NewRouter(),
		executor: executor,
		renderer: renderer,
		parser:   engine.NewSQLParser(),
	}

	s.setupRoutes()
	return s, nil
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes() {
	r := s.router

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.RealIP)

	// Static assets
	if s.config.AssetsDir != "" {
		fileServer := http.FileServer(http.Dir(s.config.AssetsDir))
		r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))
	}

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Main handler for SQL pages
	r.Get("/*", s.handleSQLPage)
	r.Post("/*", s.handleSQLPage)
}

// handleSQLPage handles requests for SQL-based pages
func (s *Server) handleSQLPage(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index"
	}

	// Find the SQL file
	sqlFile := s.findSQLFile(path)
	if sqlFile == "" {
		http.NotFound(w, r)
		return
	}

	// Parse the SQL file
	statements, err := s.parser.ParseFile(sqlFile)
	if err != nil {
		s.handleError(w, r, fmt.Errorf("failed to parse SQL file: %w", err))
		return
	}

	// Build parameters from query string and form
	params := s.extractParams(r)

	// Execute SQL statements
	ctx := r.Context()
	rows, err := s.executor.ExecuteMultiple(ctx, statements, params)
	if err != nil {
		s.handleError(w, r, fmt.Errorf("SQL execution failed: %w", err))
		return
	}

	// Check for redirects or special responses
	if s.handleSpecialResponses(w, r, rows) {
		return
	}

	// Parse components from results
	components := s.renderer.ParseComponents(rows)

	// Build page data
	pageData := map[string]interface{}{
		"Path":   path,
		"Params": params,
		"Method": r.Method,
	}

	// Render the page
	html, err := s.renderer.RenderPage(components, pageData)
	if err != nil {
		s.handleError(w, r, fmt.Errorf("render failed: %w", err))
		return
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// For HTMX, just return the content without the full layout
		// TODO: Implement partial rendering
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// findSQLFile looks for a SQL file matching the path
func (s *Server) findSQLFile(path string) string {
	// Clean the path
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		path = "index"
	}

	// Try different extensions and paths
	candidates := []string{
		filepath.Join(s.config.SQLDir, path+".sql"),
		filepath.Join(s.config.SQLDir, path, "index.sql"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// extractParams extracts parameters from the request
func (s *Server) extractParams(r *http.Request) map[string]string {
	params := make(map[string]string)

	// Query parameters
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	// Form values (for POST)
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err == nil {
			for key, values := range r.PostForm {
				if len(values) > 0 {
					params[key] = values[0]
				}
			}
		}
	}

	// URL path parameters from chi
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		for i, key := range rctx.URLParams.Keys {
			if i < len(rctx.URLParams.Values) {
				params[key] = rctx.URLParams.Values[i]
			}
		}
	}

	return params
}

// handleSpecialResponses handles redirects and other special SQL responses
func (s *Server) handleSpecialResponses(w http.ResponseWriter, r *http.Request, rows []engine.Row) bool {
	for _, row := range rows {
		// Check for redirect
		if redirect, ok := row["redirect"].(string); ok && redirect != "" {
			// For HTMX requests, use HX-Redirect header
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", redirect)
				w.WriteHeader(http.StatusOK)
			} else {
				http.Redirect(w, r, redirect, http.StatusSeeOther)
			}
			return true
		}

		// Check for JSON response
		if _, ok := row["json"]; ok {
			w.Header().Set("Content-Type", "application/json")
			// TODO: Implement JSON serialization
			return true
		}
	}
	return false
}

// handleError handles errors with appropriate response
func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Error: %v", err)

	if s.config.Debug {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	log.Printf("GoPage server starting on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.router)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.executor.Close()
}

// Router returns the underlying chi router for customization
func (s *Server) Router() *chi.Mux {
	return s.router
}
