package engine

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// SchemaInfo contient les informations sur une table
type SchemaInfo struct {
	TableName   string                    `json:"table_name"`
	Columns     []ColumnInfo              `json:"columns"`
	ForeignKeys map[string]ForeignKeyInfo `json:"foreign_keys"`
	Indexes     []IndexInfo               `json:"indexes"`
	LastUpdated time.Time                 `json:"last_updated"`
}

// ColumnInfo contient les informations sur une colonne
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	NotNull      bool   `json:"not_null"`
	DefaultValue string `json:"default_value"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// IndexInfo contient les informations sur un index
type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// SchemaCache gère le cache des schémas de base de données
type SchemaCache struct {
	db    *sql.DB
	cache map[string]*SchemaInfo
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewSchemaCache crée un nouveau cache de schéma
func NewSchemaCache(db *sql.DB, ttl time.Duration) *SchemaCache {
	return &SchemaCache{
		db:    db,
		cache: make(map[string]*SchemaInfo),
		ttl:   ttl,
	}
}

// GetSchema récupère le schéma d'une table (avec cache)
func (sc *SchemaCache) GetSchema(tableName string) (*SchemaInfo, error) {
	// Vérifier le cache en lecture
	sc.mutex.RLock()
	if cached, exists := sc.cache[tableName]; exists {
		// Vérifier si le cache est encore valide
		if time.Since(cached.LastUpdated) < sc.ttl {
			sc.mutex.RUnlock()
			return cached, nil
		}
	}
	sc.mutex.RUnlock()

	// Acquérir le lock en écriture pour rafraîchir le cache
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// Double vérification après avoir acquis le lock
	if cached, exists := sc.cache[tableName]; exists {
		if time.Since(cached.LastUpdated) < sc.ttl {
			return cached, nil
		}
	}

	// Charger le schéma depuis la base de données
	schema, err := sc.loadSchemaFromDB(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema for %s: %w", tableName, err)
	}

	// Mettre en cache
	sc.cache[tableName] = schema
	return schema, nil
}

// loadSchemaFromDB charge le schéma depuis la base de données
func (sc *SchemaCache) loadSchemaFromDB(tableName string) (*SchemaInfo, error) {
	schema := &SchemaInfo{
		TableName:   tableName,
		Columns:     []ColumnInfo{},
		ForeignKeys: make(map[string]ForeignKeyInfo),
		Indexes:     []IndexInfo{},
		LastUpdated: time.Now(),
	}

	// Charger les colonnes
	if err := sc.loadColumns(schema); err != nil {
		return nil, err
	}

	// Charger les clés étrangères
	if err := sc.loadForeignKeys(schema); err != nil {
		return nil, err
	}

	// Charger les index
	if err := sc.loadIndexes(schema); err != nil {
		return nil, err
	}

	return schema, nil
}

// loadColumns charge les informations des colonnes
func (sc *SchemaCache) loadColumns(schema *SchemaInfo) error {
	query := fmt.Sprintf("PRAGMA table_info(%s)", schema.TableName)
	rows, err := sc.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue interface{}
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}

		column := ColumnInfo{
			Name:         name,
			Type:         dataType,
			NotNull:      notNull == 1,
			IsPrimaryKey: pk == 1,
		}

		if defaultValue != nil {
			column.DefaultValue = fmt.Sprintf("%v", defaultValue)
		}

		schema.Columns = append(schema.Columns, column)
	}

	return nil
}

// loadForeignKeys charge les clés étrangères
func (sc *SchemaCache) loadForeignKeys(schema *SchemaInfo) error {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", schema.TableName)
	rows, err := sc.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, seq int
		var table, from, to string
		var onAction, onUpdate, match string

		if err := rows.Scan(&id, &seq, &table, &from, &to, &onAction, &onUpdate, &match); err != nil {
			return fmt.Errorf("failed to scan foreign key: %w", err)
		}

		schema.ForeignKeys[from] = ForeignKeyInfo{
			Name:          fmt.Sprintf("fk_%d", id),
			PrimaryTable:  table,
			PrimaryColumn: to,
			ForeignColumn: from,
		}
	}

	return nil
}

// loadIndexes charge les index
func (sc *SchemaCache) loadIndexes(schema *SchemaInfo) error {
	query := fmt.Sprintf("PRAGMA index_list(%s)", schema.TableName)
	rows, err := sc.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to get index list: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return fmt.Errorf("failed to scan index info: %w", err)
		}

		// Ignorer l'index automatique de la clé primaire
		if origin == "pk" {
			continue
		}

		// Charger les colonnes de l'index
		columns, err := sc.getIndexColumns(name)
		if err != nil {
			return fmt.Errorf("failed to get index columns for %s: %w", name, err)
		}

		schema.Indexes = append(schema.Indexes, IndexInfo{
			Name:    name,
			Columns: columns,
			Unique:  unique == 1,
		})
	}

	return nil
}

// getIndexColumns récupère les colonnes d'un index
func (sc *SchemaCache) getIndexColumns(indexName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA index_info(%s)", indexName)
	rows, err := sc.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get index info: %w", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var seq, cid int
		var name string

		if err := rows.Scan(&seq, &cid, &name); err != nil {
			return nil, fmt.Errorf("failed to scan index column: %w", err)
		}

		columns = append(columns, name)
	}

	return columns, nil
}

// InvalidateCache invalide le cache pour une table spécifique
func (sc *SchemaCache) InvalidateCache(tableName string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	delete(sc.cache, tableName)
}

// ClearCache vide tout le cache
func (sc *SchemaCache) ClearCache() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.cache = make(map[string]*SchemaInfo)
}

// GetCachedTables retourne la liste des tables en cache
func (sc *SchemaCache) GetCachedTables() []string {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	var tables []string
	for tableName := range sc.cache {
		tables = append(tables, tableName)
	}
	return tables
}

// CacheStats retourne des statistiques sur le cache
func (sc *SchemaCache) CacheStats() map[string]interface{} {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	stats := map[string]interface{}{
		"cached_tables": len(sc.cache),
		"tables":        sc.GetCachedTables(),
	}

	// Calculer l'âge du cache le plus ancien et le plus récent
	if len(sc.cache) > 0 {
		var oldest, newest time.Time
		for _, schema := range sc.cache {
			if oldest.IsZero() || schema.LastUpdated.Before(oldest) {
				oldest = schema.LastUpdated
			}
			if newest.IsZero() || schema.LastUpdated.After(newest) {
				newest = schema.LastUpdated
			}
		}
		stats["oldest_cache"] = oldest
		stats["newest_cache"] = newest
		stats["oldest_age_seconds"] = time.Since(oldest).Seconds()
		stats["newest_age_seconds"] = time.Since(newest).Seconds()
	}

	return stats
}
