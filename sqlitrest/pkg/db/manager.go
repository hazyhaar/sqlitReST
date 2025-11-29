package db

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/cl-ment/sqlitrest/pkg/config"
	_ "modernc.org/sqlite" // zombiezen utilise modernc en interne
)

type Manager struct {
	databases map[string]*Database
	mutex     sync.RWMutex
}

type Database struct {
	name    string
	Writer  *sql.DB
	Readers []*sql.DB
	path    string
	mode    string
}

func NewManager(cfg *config.Config) (*Manager, error) {
	m := &Manager{
		databases: make(map[string]*Database),
	}

	// Attacher les bases de données depuis la config
	for _, dbCfg := range cfg.Databases {
		if err := m.AttachDB(dbCfg.Name, dbCfg.Path, dbCfg.Mode); err != nil {
			return nil, fmt.Errorf("failed to attach database %s: %w", dbCfg.Name, err)
		}
	}

	return m, nil
}

func (m *Manager) AttachDB(name, path, mode string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if mode != "readwrite" && mode != "readonly" && mode != "memory" {
		return fmt.Errorf("invalid mode: %s", mode)
	}

	db := &Database{
		name: name,
		path: path,
		mode: mode,
	}

	if mode == "readwrite" {
		writer, err := m.createWriter(path)
		if err != nil {
			return err
		}
		db.Writer = writer
		db.Readers = m.createReaders(path, 5)
	} else {
		db.Readers = m.createReaders(path, 10)
	}

	m.databases[name] = db
	return nil
}

func (m *Manager) DetachDB(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if db, exists := m.databases[name]; exists {
		if db.Writer != nil {
			defer db.Writer.Close()
		}
		for _, reader := range db.Readers {
			defer reader.Close()
		}
		delete(m.databases, name)
	}
	return nil
}

func (m *Manager) GetDB(name string) (*Database, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	db, exists := m.databases[name]
	if !exists {
		return nil, fmt.Errorf("database %s not found", name)
	}
	return db, nil
}

func (m *Manager) ListDatabases() map[string]*Database {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*Database)
	for k, v := range m.databases {
		result[k] = v
	}
	return result
}

func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for name, db := range m.databases {
		if db.Writer != nil {
			db.Writer.Close()
		}
		for _, reader := range db.Readers {
			reader.Close()
		}
		delete(m.databases, name)
	}
	return nil
}

func (m *Manager) createWriter(path string) (*sql.DB, error) {
	// Utiliser sqlitex pour database/sql integration
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Pragmas SQLite
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	return db, nil
}

func (m *Manager) createReaders(path string, count int) []*sql.DB {
	readers := make([]*sql.DB, count)
	for i := 0; i < count; i++ {
		db, _ := sql.Open("sqlite", path+"?mode=ro")
		db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA foreign_keys=ON;")
		readers[i] = db
	}
	return readers
}
