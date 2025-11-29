package main

import (
	"fmt"
	"os"

	"github.com/cl-ment/sqlitrest/internal/server"
	"github.com/cl-ment/sqlitrest/pkg/auth"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "generate-token" {
		generateTestToken()
		return
	}

	fmt.Printf("SQLitREST v%s - SQLite REST API Server\n", version)

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateTestToken() {
	config := auth.JWTConfig{
		Enabled:   true,
		Algorithm: "HS256",
		Secret:    "sqlitrest-secret-key-2025",
		Issuer:    "sqlitrest",
		Audience:  []string{"sqlitrest-api"},
	}

	jwtManager, err := auth.NewJWTManager(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create JWT manager: %v\n", err)
		os.Exit(1)
	}

	// Générer token admin
	adminToken, err := jwtManager.GenerateToken("1", "admin", "", []string{"read:all", "write:all"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate admin token: %v\n", err)
		os.Exit(1)
	}

	// Générer token user
	userToken, err := jwtManager.GenerateToken("2", "user", "", []string{"read:users", "write:posts"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate user token: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Admin Token: %s\n", adminToken)
	fmt.Printf("User Token: %s\n", userToken)
}
