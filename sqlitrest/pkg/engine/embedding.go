package engine

import (
	"database/sql"
	"fmt"
	"strings"
)

// ResourceEmbedding gère l'inclusion de relations (foreign keys)
type ResourceEmbedding struct {
	db *sql.DB
}

// NewResourceEmbedding crée un nouveau gestionnaire d'embedding
func NewResourceEmbedding(db *sql.DB) *ResourceEmbedding {
	return &ResourceEmbedding{db: db}
}

// EmbeddedRelation représente une relation à inclure
type EmbeddedRelation struct {
	Name     string
	Table    string
	Column   string
	Alias    string
	Required bool
}

// ParseEmbeddedRelations parse les relations depuis le paramètre select
func ParseEmbeddedRelations(selectParam string) ([]EmbeddedRelation, error) {
	if selectParam == "*" {
		return nil, nil
	}

	parts := strings.Split(selectParam, ",")
	var relations []EmbeddedRelation

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Format: relation_name(columns) ou relation_name
		if strings.Contains(part, "(") && strings.Contains(part, ")") {
			// Format avec colonnes spécifiques: posts(title,created_at)
			relationName := part[:strings.Index(part, "(")]
			columnsPart := strings.Trim(part[strings.Index(part, "(")+1:strings.Index(part, ")")], "()")

			columns := strings.Split(columnsPart, ",")
			for i, col := range columns {
				columns[i] = strings.TrimSpace(col)
			}

			relations = append(relations, EmbeddedRelation{
				Name:  relationName,
				Alias: relationName,
			})
		} else if strings.Contains(part, ".") {
			// Format imbriqué: user.profile ou posts.comments.user
			relations = append(relations, EmbeddedRelation{
				Name:  part,
				Alias: part,
			})
		} else if part != "*" {
			// Relation simple: posts
			relations = append(relations, EmbeddedRelation{
				Name:  part,
				Alias: part,
			})
		}
	}

	return relations, nil
}

// BuildEmbeddedQuery construit une requête avec JOINs pour les relations
func (e *ResourceEmbedding) BuildEmbeddedQuery(baseQuery string, relations []EmbeddedRelation, baseTable string) (string, error) {
	if len(relations) == 0 {
		return baseQuery, nil
	}

	// Analyser les foreign keys de la table de base
	foreignKeys, err := e.GetForeignKeys(baseTable)
	if err != nil {
		return baseQuery, fmt.Errorf("failed to analyze foreign keys: %w", err)
	}

	// Construire les JOINs
	var joinClauses []string
	var selectColumns []string

	// Ajouter les colonnes de la table de base
	selectColumns = append(selectColumns, fmt.Sprintf("%s.*", baseTable))

	for _, relation := range relations {
		fkInfo, exists := foreignKeys[relation.Name]
		if !exists {
			// Pour l'instant, ignorer les relations qui ne sont pas des foreign keys directs
			continue
		}

		// Relation directe : LEFT JOIN related_table ON base_table.fk_column = related_table.id
		joinClause := fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.%s",
			fkInfo.PrimaryTable, baseTable, fkInfo.ForeignColumn, fkInfo.PrimaryTable, fkInfo.PrimaryColumn)
		joinClauses = append(joinClauses, joinClause)

		// Ajouter colonnes de la relation
		selectColumns = append(selectColumns, fmt.Sprintf("%s as %s",
			fmt.Sprintf("%s.*", fkInfo.PrimaryTable), relation.Name))
	}

	// Reconstruire la requête avec les JOINs et les nouvelles colonnes
	if strings.Contains(strings.ToUpper(baseQuery), "SELECT") {
		// Remplacer SELECT * par nos colonnes
		newQuery := strings.Replace(baseQuery, "SELECT *", fmt.Sprintf("SELECT %s", strings.Join(selectColumns, ", ")), 1)

		// Insérer les JOINs après FROM mais avant WHERE
		fromIndex := strings.Index(strings.ToUpper(newQuery), " FROM ")
		if fromIndex != -1 {
			afterFrom := newQuery[fromIndex:]
			insertPos := len(afterFrom)

			// Trouver où insérer les JOINs (avant WHERE, ORDER BY, etc.)
			clauses := []string{" WHERE ", " ORDER BY ", " GROUP BY ", " HAVING ", " LIMIT ", " OFFSET "}
			for _, clause := range clauses {
				if pos := strings.Index(strings.ToUpper(afterFrom), clause); pos != -1 && pos < insertPos {
					insertPos = pos
				}
			}

			if insertPos == len(afterFrom) {
				// Ajouter à la fin
				newQuery = newQuery + " " + strings.Join(joinClauses, " ")
			} else {
				// Insérer avant la clause trouvée
				beforeClause := afterFrom[:insertPos]
				afterClause := afterFrom[insertPos:]
				newQuery = newQuery[:fromIndex] + beforeClause + " " + strings.Join(joinClauses, " ") + afterClause
			}
		}

		return newQuery, nil
	}

	return baseQuery, nil
}

// GetForeignKeys récupère les clés étrangères d'une table
func (e *ResourceEmbedding) GetForeignKeys(tableName string) (map[string]ForeignKeyInfo, error) {
	// Version simplifiée qui fonctionne avec SQLite
	// PRAGMA foreign_key_list ne supporte pas les paramètres, on construit la requête directement
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", tableName)

	rows, err := e.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer rows.Close()

	foreignKeys := make(map[string]ForeignKeyInfo)
	for rows.Next() {
		var id int
		var seq int
		var table string
		var from string
		var to string
		var onAction string
		var onUpdate string
		var match string

		if err := rows.Scan(&id, &seq, &table, &from, &to, &onAction, &onUpdate, &match); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		foreignKeys[from] = ForeignKeyInfo{
			Name:          fmt.Sprintf("fk_%d", id),
			PrimaryTable:  table,
			PrimaryColumn: to,
			ForeignColumn: from,
		}
	}

	return foreignKeys, nil
}

// ForeignKeyInfo contient les informations sur une clé étrangère
type ForeignKeyInfo struct {
	Name          string
	PrimaryTable  string
	PrimaryColumn string
	ForeignColumn string
}
