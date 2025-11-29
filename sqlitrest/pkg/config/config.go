package config

import (
	"fmt"
	"os"

	"github.com/cl-ment/sqlitrest/pkg/auth"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server    ServerConfig     `toml:"server"`
	Databases []DatabaseConfig `toml:"databases"`
	Auth      AuthConfig       `toml:"auth"`
}

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type DatabaseConfig struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
	Mode string `toml:"mode"`
}

type AuthConfig struct {
	JWT auth.JWTConfig `toml:"jwt"`
}

type JWTConfig struct {
	Enabled   bool   `toml:"enabled"`
	Algorithm string `toml:"algorithm"`
	Secret    string `toml:"secret"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Databases: []DatabaseConfig{
			{
				Name: "main",
				Path: "./data/main.db",
				Mode: "readwrite",
			},
		},
		Auth: AuthConfig{
			JWT: auth.JWTConfig{
				Enabled:   false,
				Algorithm: "HS256",
				Secret:    "change-me",
			},
		},
	}

	// Essayer de charger depuis fichier
	if data, err := os.ReadFile("sqlitrest.toml"); err == nil {
		if err := toml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}

	return cfg, nil
}
