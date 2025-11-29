package engine

import (
	"database/sql"
	"fmt"
	"strings"
)

// QueryResult représente le résultat d'une requête SQL
type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Count   int                      `json:"count"`
}

// SQLBuilder construit des requêtes SQL à partir des paramètres parsés
type SQLBuilder struct{}

// NewSQLBuilder crée un nouveau builder
func NewSQLBuilder() *SQLBuilder {
	return &SQLBuilder{}
}

// BuildSelect construit une requête SELECT complète
func (b *SQLBuilder) BuildSelect(params *QueryParameters) (string, []interface{}, error) {
	var args []interface{}

	// Construction de la clause SELECT
	selectClause := b.buildSelectClause(params.Select)

	// Construction de la clause FROM
	fromClause := fmt.Sprintf("FROM %s", b.quoteIdentifier(params.Table))

	// Construction de la clause WHERE
	whereClause, whereArgs := b.buildWhereClause(params)
	args = append(args, whereArgs...)

	// Construction de la clause ORDER BY
	orderClause := b.buildOrderClause(params.Order)

	// Construction de la clause LIMIT/OFFSET
	limitClause, limitArgs := b.buildLimitClause(params.Limit, params.Offset)
	args = append(args, limitArgs...)

	// Assemblage final
	query := fmt.Sprintf("SELECT %s %s %s %s %s",
		selectClause, fromClause, whereClause, orderClause, limitClause)

	// Nettoyage des espaces multiples
	query = strings.TrimSpace(query)
	query = strings.ReplaceAll(query, "  ", " ")

	return query, args, nil
}

// BuildSelectWithEmbedding construit une requête SELECT avec resource embedding
func (b *SQLBuilder) BuildSelectWithEmbedding(params *QueryParameters, embedding *ResourceEmbedding) (string, []interface{}, error) {
	// Construire la requête de base
	query, args, err := b.BuildSelect(params)
	if err != nil {
		return "", nil, err
	}

	// Parser les relations depuis select - convertir []string en string
	selectStr := "*"
	if len(params.Select) > 0 {
		selectStr = strings.Join(params.Select, ",")
	}

	relations, err := ParseEmbeddedRelations(selectStr)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse embedded relations: %w", err)
	}

	// Appliquer l'embedding si nécessaire
	if len(relations) > 0 && embedding != nil {
		embeddedQuery, err := embedding.BuildEmbeddedQuery(query, relations, params.Table)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build embedded query: %w", err)
		}
		return embeddedQuery, args, nil
	}

	return query, args, nil
}

// BuildInsert construit une requête INSERT
func (b *SQLBuilder) BuildInsert(table string, data map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data provided for insert")
	}

	var columns []string
	var placeholders []string
	var args []interface{}

	for column, value := range data {
		columns = append(columns, b.quoteIdentifier(column))
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		b.quoteIdentifier(table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	return query, args, nil
}

// BuildUpdate construit une requête UPDATE
func (b *SQLBuilder) BuildUpdate(table string, data map[string]interface{}, filters []Filter) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("no data provided for update")
	}

	var setParts []string
	var args []interface{}

	for column, value := range data {
		setParts = append(setParts, fmt.Sprintf("%s = ?", b.quoteIdentifier(column)))
		args = append(args, value)
	}

	// Construction de la clause WHERE
	whereClause, whereArgs := b.buildWhereClauseFromFilters(filters)
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s %s",
		b.quoteIdentifier(table),
		strings.Join(setParts, ", "),
		whereClause)

	return query, args, nil
}

// BuildDelete construit une requête DELETE
func (b *SQLBuilder) BuildDelete(table string, filters []Filter) (string, []interface{}, error) {
	whereClause, args := b.buildWhereClauseFromFilters(filters)

	query := fmt.Sprintf("DELETE FROM %s %s", b.quoteIdentifier(table), whereClause)
	return query, args, nil
}

// buildLogicalCondition construit une condition logique (and/or)
func (b *SQLBuilder) buildLogicalCondition(condition LogicalCondition) (string, []interface{}) {
	if len(condition.Filters) == 0 {
		return "", []interface{}{}
	}

	var subConditions []string
	var args []interface{}

	// Construire chaque sous-condition
	for _, filter := range condition.Filters {
		filterCondition, filterArgs := b.buildFilterCondition(filter)
		subConditions = append(subConditions, filterCondition)
		args = append(args, filterArgs...)
	}

	// Combiner avec AND ou OR
	operator := "AND"
	if condition.Type == "or" {
		operator = "OR"
	}

	if len(subConditions) == 1 {
		return subConditions[0], args
	}

	return fmt.Sprintf("(%s)", strings.Join(subConditions, fmt.Sprintf(" %s ", operator))), args
}

// buildSelectClause construit la clause SELECT
func (b *SQLBuilder) buildSelectClause(selectColumns []string) string {
	if len(selectColumns) == 0 {
		return "*"
	}

	var quoted []string
	for _, col := range selectColumns {
		quoted = append(quoted, b.quoteIdentifier(col))
	}

	return strings.Join(quoted, ", ")
}

// buildWhereClause construit la clause WHERE avec support des conditions logiques
func (b *SQLBuilder) buildWhereClause(params *QueryParameters) (string, []interface{}) {
	var allConditions []string
	var allArgs []interface{}

	// Ajouter les filtres simples
	if len(params.Filters) > 0 {
		simpleWhere, simpleArgs := b.buildWhereClauseFromFilters(params.Filters)
		if simpleWhere != "" {
			allConditions = append(allConditions, strings.TrimPrefix(simpleWhere, "WHERE "))
			allArgs = append(allArgs, simpleArgs...)
		}
	}

	// Ajouter les conditions logiques (and/or)
	for _, condition := range params.Conditions {
		logicalWhere, logicalArgs := b.buildLogicalCondition(condition)
		if logicalWhere != "" {
			allConditions = append(allConditions, logicalWhere)
			allArgs = append(allArgs, logicalArgs...)
		}
	}

	if len(allConditions) == 0 {
		return "", []interface{}{}
	}

	return fmt.Sprintf("WHERE %s", strings.Join(allConditions, " AND ")), allArgs
}

// buildWhereClauseFromFilters construit la clause WHERE depuis les filtres
func (b *SQLBuilder) buildWhereClauseFromFilters(filters []Filter) (string, []interface{}) {
	if len(filters) == 0 {
		return "", []interface{}{}
	}

	var conditions []string
	var args []interface{}

	for _, filter := range filters {
		condition, filterArgs := b.buildFilterCondition(filter)
		conditions = append(conditions, condition)
		args = append(args, filterArgs...)
	}

	if len(conditions) == 0 {
		return "", []interface{}{}
	}

	return fmt.Sprintf("WHERE %s", strings.Join(conditions, " AND ")), args
}

// buildFilterCondition construit une condition de filtre
func (b *SQLBuilder) buildFilterCondition(filter Filter) (string, []interface{}) {
	column := b.quoteIdentifier(filter.Column)
	operator := filter.Operator
	value := filter.Value

	switch operator {
	case OpEqual:
		return fmt.Sprintf("%s = ?", column), []interface{}{value}
	case OpNotEqual:
		return fmt.Sprintf("%s != ?", column), []interface{}{value}
	case OpGreaterThan:
		return fmt.Sprintf("%s > ?", column), []interface{}{value}
	case OpGreaterEqual:
		return fmt.Sprintf("%s >= ?", column), []interface{}{value}
	case OpLessThan:
		return fmt.Sprintf("%s < ?", column), []interface{}{value}
	case OpLessEqual:
		return fmt.Sprintf("%s <= ?", column), []interface{}{value}
	case OpLike:
		return fmt.Sprintf("%s LIKE ?", column), []interface{}{value}
	case OpILike:
		return fmt.Sprintf("%s ILIKE ?", column), []interface{}{value}
	case OpIn:
		// Gérer les valeurs multiples (séparées par des virgules)
		values := strings.Split(value, ",")
		placeholders := make([]string, len(values))
		args := make([]interface{}, len(values))

		for i, v := range values {
			placeholders[i] = "?"
			args[i] = strings.TrimSpace(v)
		}

		return fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", ")), args
	case OpIs:
		if strings.ToLower(value) == "null" {
			return fmt.Sprintf("%s IS NULL", column), []interface{}{}
		}
		return fmt.Sprintf("%s IS ?", column), []interface{}{value}
	case OpNot:
		// Opérateur NOT générique
		if strings.ToLower(value) == "null" {
			return fmt.Sprintf("%s IS NOT NULL", column), []interface{}{}
		}
		return fmt.Sprintf("%s != ?", column), []interface{}{value}
	default:
		return fmt.Sprintf("%s = ?", column), []interface{}{value}
	}
}

// buildOrderClause construit la clause ORDER BY
func (b *SQLBuilder) buildOrderClause(orderClauses []OrderClause) string {
	if len(orderClauses) == 0 {
		return ""
	}

	var parts []string
	for _, order := range orderClauses {
		direction := strings.ToUpper(order.Direction)
		if direction != "ASC" && direction != "DESC" {
			direction = "ASC"
		}
		parts = append(parts, fmt.Sprintf("%s %s", b.quoteIdentifier(order.Column), direction))
	}

	return fmt.Sprintf("ORDER BY %s", strings.Join(parts, ", "))
}

// buildLimitClause construit les clauses LIMIT et OFFSET
func (b *SQLBuilder) buildLimitClause(limit *int, offset *int) (string, []interface{}) {
	var parts []string
	var args []interface{}

	if limit != nil {
		parts = append(parts, "LIMIT ?")
		args = append(args, *limit)
	}

	if offset != nil {
		parts = append(parts, "OFFSET ?")
		args = append(args, *offset)
	}

	return strings.Join(parts, " "), args
}

// quoteIdentifier protège les identifiants SQL avec des backticks
func (b *SQLBuilder) quoteIdentifier(name string) string {
	// Échapper les backticks existants
	escaped := strings.ReplaceAll(name, "`", "``")
	return fmt.Sprintf("`%s`", escaped)
}

// Executor exécute les requêtes SQL construites
type Executor struct {
	db *sql.DB
}

// NewExecutor crée un nouvel exécuteur
func NewExecutor(db *sql.DB) *Executor {
	return &Executor{db: db}
}

// ExecuteSelect exécute une requête SELECT
func (e *Executor) ExecuteSelect(query string, args []interface{}) (*QueryResult, error) {
	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Récupérer les noms de colonnes
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Lire les données
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Construire la map pour cette ligne
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &QueryResult{
		Rows:    results,
		Columns: columns,
		Count:   len(results),
	}, nil
}

// ExecuteCommand exécute une requête INSERT/UPDATE/DELETE
func (e *Executor) ExecuteCommand(query string, args []interface{}) (*QueryResult, error) {
	result, err := e.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return &QueryResult{
		Rows: []map[string]interface{}{
			{"rows_affected": rowsAffected},
		},
		Columns: []string{"rows_affected"},
		Count:   1,
	}, nil
}
