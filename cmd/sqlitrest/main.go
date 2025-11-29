package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/hazyhaar/sqlitrest/pkg/server"
)

var (
	version = "0.1.0"
)

func main() {
	// Command-line flags
	port := flag.Int("port", 8080, "Port to listen on")
	dbPath := flag.String("db", "./data.db", "Path to SQLite database")
	sqlDir := flag.String("sql", "./sql", "Directory containing SQL pages")
	templateDir := flag.String("templates", "./templates", "Directory containing HTML templates")
	assetsDir := flag.String("assets", "./assets", "Directory for static assets")
	debug := flag.Bool("debug", false, "Enable debug mode")
	showVersion := flag.Bool("version", false, "Show version and exit")

	// LLM configuration
	llmEndpoint := flag.String("llm-endpoint", "", "LLM API endpoint (default: Cerebras)")
	llmAPIKey := flag.String("llm-api-key", "", "LLM API key (or use LLM_API_KEY env)")
	llmModel := flag.String("llm-model", "", "LLM model name (default: llama-3.3-70b)")

	// HTTP configuration
	httpTimeout := flag.Int("http-timeout", 30, "Timeout for HTTP functions (seconds)")

	flag.Parse()

	if *showVersion {
		fmt.Printf("sqlitREST version %s\n", version)
		os.Exit(0)
	}

	// Resolve paths relative to working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Get LLM API key from flag or environment
	apiKey := *llmAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("LLM_API_KEY")
	}
	// Also check CEREBRAS_API_KEY for convenience
	if apiKey == "" {
		apiKey = os.Getenv("CEREBRAS_API_KEY")
	}

	cfg := server.Config{
		Port:        *port,
		DBPath:      resolvePath(workDir, *dbPath),
		SQLDir:      resolvePath(workDir, *sqlDir),
		TemplateDir: resolvePath(workDir, *templateDir),
		AssetsDir:   resolvePath(workDir, *assetsDir),
		Debug:       *debug,
		LLMEndpoint: *llmEndpoint,
		LLMAPIKey:   apiKey,
		LLMModel:    *llmModel,
		HTTPTimeout: *httpTimeout,
	}

	// Ensure required directories exist
	if err := ensureDir(cfg.SQLDir); err != nil {
		log.Fatalf("SQL directory error: %v", err)
	}

	// Create and configure server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
		cancel()
		os.Exit(0)
	}()

	// Print startup info
	printStartupInfo(cfg)

	// Run server
	if err := srv.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func resolvePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func printStartupInfo(cfg server.Config) {
	log.Println("=================================")
	log.Printf("  sqlitREST v%s", version)
	log.Println("=================================")
	log.Printf("  Port:      %d", cfg.Port)
	log.Printf("  Database:  %s", cfg.DBPath)
	log.Printf("  SQL Dir:   %s", cfg.SQLDir)
	log.Printf("  Templates: %s", cfg.TemplateDir)
	log.Printf("  Assets:    %s", cfg.AssetsDir)
	log.Printf("  Debug:     %v", cfg.Debug)
	if cfg.LLMAPIKey != "" {
		log.Printf("  LLM:       enabled")
		if cfg.LLMModel != "" {
			log.Printf("  LLM Model: %s", cfg.LLMModel)
		}
	} else {
		log.Printf("  LLM:       disabled (no API key)")
	}
	log.Printf("  HTTP Timeout: %ds", cfg.HTTPTimeout)
	log.Println("=================================")
}
