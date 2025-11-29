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

	"github.com/horos/gopage/pkg/server"
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

	flag.Parse()

	if *showVersion {
		fmt.Printf("GoPage version %s\n", version)
		os.Exit(0)
	}

	// Resolve paths relative to working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	cfg := server.Config{
		Port:        *port,
		DBPath:      resolvePath(workDir, *dbPath),
		SQLDir:      resolvePath(workDir, *sqlDir),
		TemplateDir: resolvePath(workDir, *templateDir),
		AssetsDir:   resolvePath(workDir, *assetsDir),
		Debug:       *debug,
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
	log.Printf("  GoPage v%s", version)
	log.Println("=================================")
	log.Printf("  Port:      %d", cfg.Port)
	log.Printf("  Database:  %s", cfg.DBPath)
	log.Printf("  SQL Dir:   %s", cfg.SQLDir)
	log.Printf("  Templates: %s", cfg.TemplateDir)
	log.Printf("  Assets:    %s", cfg.AssetsDir)
	log.Printf("  Debug:     %v", cfg.Debug)
	log.Println("=================================")
}
