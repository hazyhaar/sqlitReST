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
	r.Put("/*", s.handleSQLPage)
	r.Delete("/*", s.handleSQLPage)
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

	// Parse request parameters with pagination support
	reqParams := ParseRequestParams(r)
	params := reqParams.ToSQLParams()

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

	// Parse components from results (using data-aware parser)
	components := s.renderer.ParseDataComponents(rows)

	// Build page data
	pageData := map[string]interface{}{
		"Path":       path,
		"Params":     params,
		"Method":     r.Method,
		"Page":       reqParams.Page,
		"PerPage":    reqParams.PerPage,
		"Sort":       reqParams.Sort,
		"Search":     reqParams.Search,
		"IsHTMX":     r.Header.Get("HX-Request") == "true",
		"RequestURL": r.URL.String(),
	}

	// Check if this is an HTMX request for partial content
	isHTMX := r.Header.Get("HX-Request") == "true"
	htmxTarget := r.Header.Get("HX-Target")

	// Render the page
	var html string
	if isHTMX && htmxTarget != "" {
		// Partial rendering for HTMX - only render content
		html, err = s.renderer.RenderPartial(components, pageData)
	} else {
		html, err = s.renderer.RenderPage(components, pageData)
	}

	if err != nil {
		s.handleError(w, r, fmt.Errorf("render failed: %w", err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Add HTMX-specific headers if needed
	if isHTMX {
		// Allow pushing URL to browser history
		w.Header().Set("HX-Push-Url", r.URL.String())
	}

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


// handleSpecialResponses handles redirects and other special SQL responses
func (s *Server) handleSpecialResponses(w http.ResponseWriter, r *http.Request, rows []engine.Row) bool {
	resp := ParseSQLResponse(rows)
	return s.HandleSQLResponse(w, r, resp)
}

// handleError handles errors with appropriate response
func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Error: %v", err)

	isHTMX := r.Header.Get("HX-Request") == "true"
	message := "Une erreur est survenue"
	if s.config.Debug {
		message = err.Error()
	}

	if isHTMX {
		// Return error as HTML fragment for HTMX
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(ErrorFragment(message)))
	} else {
		http.Error(w, message, http.StatusInternalServerError)
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
