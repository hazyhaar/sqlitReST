package server

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cl-ment/sqlitrest/internal/router"
	"github.com/cl-ment/sqlitrest/pkg/config"
	"github.com/cl-ment/sqlitrest/pkg/db"
)

func Start() error {
	// Charger configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialiser DB manager
	dbManager, err := db.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create DB manager: %w", err)
	}
	defer dbManager.Close()

	// Créer router
	r := router.New(dbManager, cfg)

	// Démarrer serveur HTTP
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Starting SQLitREST on %s", addr)
		if err := r.Start(addr); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Attendre signal shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down SQLitREST...")
	return nil
}
