package openapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// OpenAPIDoc représente la structure OpenAPI 3.0
type OpenAPIDoc struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]interface{} `json:"paths"`
	Components Components             `json:"components"`
}

// Info contient les métadonnées de l'API
type Info struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Version     string  `json:"version"`
	Contact     Contact `json:"contact,omitempty"`
	License     License `json:"license,omitempty"`
}

type Contact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string      `json:"type"`
	Format      string      `json:"format,omitempty"`
	Description string      `json:"description,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// OpenAPIGenerator génère la spécification OpenAPI depuis le schéma DB
type OpenAPIGenerator struct {
	db *sql.DB
}

// NewOpenAPIGenerator crée un nouveau générateur OpenAPI
func NewOpenAPIGenerator(db *sql.DB) *OpenAPIGenerator {
	return &OpenAPIGenerator{db: db}
}

// Generate génère la spécification OpenAPI complète
func (g *OpenAPIGenerator) Generate() (*OpenAPIDoc, error) {
	// Récupérer les tables
	tables, err := g.getTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// Générer les schémas
	schemas := make(map[string]Schema)
	for _, table := range tables {
		schema, err := g.generateTableSchema(table)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for %s: %w", table, err)
		}
		schemas[table] = *schema
	}

	// Générer les paths
	paths, err := g.generatePaths(tables)
	if err != nil {
		return nil, fmt.Errorf("failed to generate paths: %w", err)
	}

	doc := &OpenAPIDoc{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       "SQLitREST API",
			Description: "RESTful API for SQLite database with PostgREST compatibility",
			Version:     "1.0.0",
			Contact: Contact{
				Name:  "SQLitREST Team",
				Email: "info@sqlitrest.com",
			},
			License: License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []Server{
			{
				URL:         "http://localhost:34334",
				Description: "Development server",
			},
		},
		Paths:      paths,
		Components: Components{Schemas: schemas},
	}

	return doc, nil
}

// getTables récupère la liste des tables
func (g *OpenAPIGenerator) getTables() ([]string, error) {
	rows, err := g.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE '_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// generateTableSchema génère le schéma pour une table
func (g *OpenAPIGenerator) generateTableSchema(tableName string) (*Schema, error) {
	rows, err := g.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	properties := make(map[string]Property)
	var required []string

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}

		property := Property{
			Type:        g.mapSQLiteTypeToOpenAPI(dataType),
			Description: fmt.Sprintf("Column %s of type %s", name, dataType),
		}

		// Ajouter des exemples basés sur le type
		switch g.mapSQLiteTypeToOpenAPI(dataType) {
		case "string":
			property.Example = "example text"
		case "integer":
			property.Example = 42
		case "number":
			property.Example = 3.14
		case "boolean":
			property.Example = true
		}

		properties[name] = property

		if notNull == 1 || pk == 1 {
			required = append(required, name)
		}
	}

	return &Schema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}, nil
}

// mapSQLiteTypeToOpenAPI convertit les types SQLite vers OpenAPI
func (g *OpenAPIGenerator) mapSQLiteTypeToOpenAPI(sqliteType string) string {
	sqliteType = strings.ToUpper(sqliteType)

	if strings.Contains(sqliteType, "INT") {
		return "integer"
	} else if strings.Contains(sqliteType, "TEXT") || strings.Contains(sqliteType, "CHAR") || strings.Contains(sqliteType, "VARCHAR") {
		return "string"
	} else if strings.Contains(sqliteType, "REAL") || strings.Contains(sqliteType, "FLOAT") || strings.Contains(sqliteType, "DOUBLE") {
		return "number"
	} else if strings.Contains(sqliteType, "BLOB") {
		return "string" // BLOB encodé en base64
	} else if strings.Contains(sqliteType, "BOOLEAN") {
		return "boolean"
	}

	return "string" // Par défaut
}

// generatePaths génère les paths OpenAPI pour les tables
func (g *OpenAPIGenerator) generatePaths(tables []string) (map[string]interface{}, error) {
	paths := make(map[string]interface{})

	for _, table := range tables {
		tablePath := fmt.Sprintf("/%s", table)

		// Path item pour cette table
		pathItem := make(map[string]interface{})

		// GET operation
		pathItem["get"] = g.generateGetOperation(table)

		// POST operation
		pathItem["post"] = g.generatePostOperation(table)

		// Item path (/{id})
		itemPath := fmt.Sprintf("/%s/{id}", table)
		itemPathItem := make(map[string]interface{})

		itemPathItem["get"] = g.generateGetItemOperation(table)
		itemPathItem["patch"] = g.generatePatchOperation(table)
		itemPathItem["delete"] = g.generateDeleteOperation(table)

		paths[tablePath] = pathItem
		paths[itemPath] = itemPathItem
	}

	return paths, nil
}

// generateGetOperation génère l'opération GET pour une collection
func (g *OpenAPIGenerator) generateGetOperation(table string) map[string]interface{} {
	return map[string]interface{}{
		"summary":     fmt.Sprintf("List %s", table),
		"description": fmt.Sprintf("Retrieve a list of %s records", table),
		"parameters": []map[string]interface{}{
			{
				"name":        "select",
				"in":          "query",
				"description": "Columns to select",
				"required":    false,
				"schema": map[string]string{
					"type": "string",
				},
			},
			{
				"name":        "order",
				"in":          "query",
				"description": "Ordering",
				"required":    false,
				"schema": map[string]string{
					"type": "string",
				},
			},
			{
				"name":        "limit",
				"in":          "query",
				"description": "Limit results",
				"required":    false,
				"schema": map[string]string{
					"type": "integer",
				},
			},
			{
				"name":        "offset",
				"in":          "query",
				"description": "Offset results",
				"required":    false,
				"schema": map[string]string{
					"type": "integer",
				},
			},
		},
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "Successful response",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "array",
							"items": map[string]string{
								"$ref": fmt.Sprintf("#/components/schemas/%s", table),
							},
						},
					},
				},
			},
		},
	}
}

// generatePostOperation génère l'opération POST
func (g *OpenAPIGenerator) generatePostOperation(table string) map[string]interface{} {
	return map[string]interface{}{
		"summary":     fmt.Sprintf("Create %s", table),
		"description": fmt.Sprintf("Create a new %s record", table),
		"requestBody": map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]string{
						"$ref": fmt.Sprintf("#/components/schemas/%s", table),
					},
				},
			},
		},
		"responses": map[string]interface{}{
			"201": map[string]interface{}{
				"description": "Created successfully",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]string{
							"$ref": fmt.Sprintf("#/components/schemas/%s", table),
						},
					},
				},
			},
		},
	}
}

// generateGetItemOperation génère l'opération GET pour un item
func (g *OpenAPIGenerator) generateGetItemOperation(table string) map[string]interface{} {
	return map[string]interface{}{
		"summary":     fmt.Sprintf("Get %s by ID", table),
		"description": fmt.Sprintf("Retrieve a specific %s record", table),
		"parameters": []map[string]interface{}{
			{
				"name":        "id",
				"in":          "path",
				"required":    true,
				"description": "Record ID",
				"schema": map[string]string{
					"type": "integer",
				},
			},
		},
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "Successful response",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]string{
							"$ref": fmt.Sprintf("#/components/schemas/%s", table),
						},
					},
				},
			},
			"404": map[string]interface{}{
				"description": "Record not found",
			},
		},
	}
}

// generatePatchOperation génère l'opération PATCH
func (g *OpenAPIGenerator) generatePatchOperation(table string) map[string]interface{} {
	return map[string]interface{}{
		"summary":     fmt.Sprintf("Update %s", table),
		"description": fmt.Sprintf("Update a specific %s record", table),
		"parameters": []map[string]interface{}{
			{
				"name":        "id",
				"in":          "path",
				"required":    true,
				"description": "Record ID",
				"schema": map[string]string{
					"type": "integer",
				},
			},
		},
		"requestBody": map[string]interface{}{
			"required": true,
			"content": map[string]interface{}{
				"application/json": map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]string{
							"$ref": fmt.Sprintf("#/components/schemas/%s", table),
						},
					},
				},
			},
		},
		"responses": map[string]interface{}{
			"200": map[string]interface{}{
				"description": "Updated successfully",
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]string{
							"$ref": fmt.Sprintf("#/components/schemas/%s", table),
						},
					},
				},
			},
			"404": map[string]interface{}{
				"description": "Record not found",
			},
		},
	}
}

// generateDeleteOperation génère l'opération DELETE
func (g *OpenAPIGenerator) generateDeleteOperation(table string) map[string]interface{} {
	return map[string]interface{}{
		"summary":     fmt.Sprintf("Delete %s", table),
		"description": fmt.Sprintf("Delete a specific %s record", table),
		"parameters": []map[string]interface{}{
			{
				"name":        "id",
				"in":          "path",
				"required":    true,
				"description": "Record ID",
				"schema": map[string]string{
					"type": "integer",
				},
			},
		},
		"responses": map[string]interface{}{
			"204": map[string]interface{}{
				"description": "Deleted successfully",
			},
			"404": map[string]interface{}{
				"description": "Record not found",
			},
		},
	}
}

// ToJSON convertit la spécification en JSON
func (doc *OpenAPIDoc) ToJSON() ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}
