//go:build mage

package main

import (
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os"
)

var Default = Build

// Build compilation optimisée natif
func Build() error {
	fmt.Println("Building sqlitrest...")

	// Build statique pour portabilité
	return sh.Run("go", "build",
		"-ldflags", "-s -w -X main.version=1.0.0",
		"-tags", "netgo,osusergo",
		"-o", "bin/sqlitrest",
		"./cmd/sqlitrest")
}

// BuildLinux build pour Linux
func BuildLinux() error {
	fmt.Println("Building for Linux...")
	return sh.Run("go", "build",
		"-ldflags", "-s -w",
		"-tags", "netgo,osusergo",
		"-o", "bin/sqlitrest-linux-amd64",
		"./cmd/sqlitrest")
}

// Test complet avec race detection
func Test() error {
	mg.Deps(TestUnit, TestRace, TestCoverage)
	return nil
}

func TestUnit() error {
	return sh.Run("go", "test", "-v", "./...")
}

func TestRace() error {
	return sh.Run("go", "test", "-race", "-v", "./...")
}

func TestCoverage() error {
	return sh.Run("go", "test", "-cover", "-coverprofile=coverage.out", "./...")
}

// Lint avec golangci-lint
func Lint() error {
	return sh.Run("golangci-lint", "run")
}

// Run développement avec reload
func Run() error {
	return sh.Run("go", "run", "./cmd/sqlitrest",
		"-config", "sqlitrest.toml",
		"-debug")
}

// Install installation système
func Install() error {
	mg.Deps(Build)

	// Installation binaire système
	return sh.Run("sudo", "cp", "bin/sqlitrest", "/usr/local/bin/")
}

// Service création service systemd
func Service() error {
	service := `[Unit]
Description=SQLitREST API Server
After=network.target

[Service]
Type=simple
User=sqlitrest
ExecStart=/usr/local/bin/sqlitrest -config /etc/sqlitrest/sqlitrest.toml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target`

	os.WriteFile("sqlitrest.service", []byte(service), 0644)
	return sh.Run("sudo", "cp", "sqlitrest.service", "/etc/systemd/system/")
}

// Clean nettoyage build
func Clean() error {
	return sh.Run("rm", "-rf", "bin/")
}

// Release package complet
func Release() error {
	mg.Deps(Build, BuildLinux, Test, Lint)

	// Création package
	os.MkdirAll("release/v1.0.0", 0755)
	sh.Run("cp", "bin/sqlitrest", "release/v1.0.0/")
	sh.Run("cp", "bin/sqlitrest-linux-amd64", "release/v1.0.0/")
	sh.Run("cp", "sqlitrest.toml.example", "release/v1.0.0/")
	sh.Run("cp", "-r", "docs/", "release/v1.0.0/")

	// Archive
	return sh.Run("tar", "-czf", "release/sqlitrest-v1.0.0.tar.gz",
		"-C", "release", "v1.0.0")
}
