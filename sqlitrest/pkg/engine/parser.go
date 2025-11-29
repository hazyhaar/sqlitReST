package engine

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// FilterOperator représente les opérateurs de filtrage PostgREST
type FilterOperator string

const (
	OpEqual        FilterOperator = "eq"
	OpNotEqual     FilterOperator = "neq"
	OpGreaterThan  FilterOperator = "gt"
	OpGreaterEqual FilterOperator = "gte"
	OpLessThan     FilterOperator = "lt"
	OpLessEqual    FilterOperator = "lte"
	OpLike         FilterOperator = "like"
	OpILike        FilterOperator = "ilike"
	OpIn           FilterOperator = "in"
	OpIs           FilterOperator = "is"
	OpNot          FilterOperator = "not"
	OpAnd          FilterOperator = "and"
	OpOr           FilterOperator = "or"
)

// Filter représente un filtre individuel
type Filter struct {
	Column   string
	Operator FilterOperator
	Value    string
}

// QueryParameters contient les paramètres parsés de la requête HTTP
type QueryParameters struct {
	Filters    []Filter
	Select     []string
	Order      []OrderClause
	Limit      *int
	Offset     *int
	Table      string
	Conditions []LogicalCondition
}

// OrderClause représente une clause de tri
type OrderClause struct {
	Column    string
	Direction string // "asc" ou "desc"
	Nulls     string // "first" ou "last"
}

// LogicalCondition représente une condition logique complexe (AND/OR)
type LogicalCondition struct {
	Type       string // "and" ou "or"
	Filters    []Filter
	Conditions []LogicalCondition
	Negated    bool
}

// QueryParser parse les paramètres HTTP en structures SQL
type QueryParser struct{}

// NewQueryParser crée un nouveau parser
func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

// ParseQuery parse une URL et ses paramètres
func (p *QueryParser) ParseQuery(path string, queryParams url.Values) (*QueryParameters, error) {
	// Extraire le nom de la table depuis le path
	table := p.extractTableFromPath(path)
	if table == "" {
		return nil, fmt.Errorf("no table found in path: %s", path)
	}

	params := &QueryParameters{
		Table:   table,
		Filters: []Filter{},
		Select:  []string{},
		Order:   []OrderClause{},
	}

	// Parser les filtres simples
	for key, values := range queryParams {
		if len(values) == 0 {
			continue
		}

		value := values[0] // Prendre la première valeur

		if p.isFilterOperator(key) {
			filter, err := p.parseFilter(key, value)
			if err != nil {
				return nil, fmt.Errorf("invalid filter %s=%s: %w", key, value, err)
			}
			params.Filters = append(params.Filters, filter)
		} else if p.isPostgRESTFilter(key) {
			// Format PostgREST: column=operator.value
			filter, err := p.parsePostgRESTFilter(key, value)
			if err != nil {
				return nil, fmt.Errorf("invalid PostgREST filter %s=%s: %w", key, value, err)
			}
			params.Filters = append(params.Filters, filter)
		} else if key == "select" {
			params.Select = p.parseSelect(value)
		} else if key == "order" {
			orderClauses, err := p.parseOrder(value)
			if err != nil {
				return nil, fmt.Errorf("invalid order clause: %w", err)
			}
			params.Order = orderClauses
		} else if key == "limit" {
			limit, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid limit: %s", value)
			}
			params.Limit = &limit
		} else if key == "offset" {
			offset, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid offset: %s", value)
			}
			params.Offset = &offset
		} else if key == "or" || key == "and" {
			condition, err := p.parseLogicalCondition(value)
			if err != nil {
				return nil, fmt.Errorf("invalid logical condition: %w", err)
			}
			condition.Type = key
			params.Conditions = append(params.Conditions, condition)
		}
	}

	return params, nil
}

// extractTableFromPath extrait le nom de la table depuis le chemin URL
func (p *QueryParser) extractTableFromPath(path string) string {
	// Format attendu: /dbname/tablename ou /tablename
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	// Si on a /dbname/tablename, prendre la dernière partie
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}

	return parts[0]
}

// isFilterOperator vérifie si la clé est un opérateur de filtre
func (p *QueryParser) isFilterOperator(key string) bool {
	return strings.Contains(key, ".") && !strings.HasPrefix(key, "order.") && !strings.HasPrefix(key, "select.") && key != "limit" && key != "offset" && key != "and" && key != "or"
}

// parseFilter parse un filtre individuel
func (p *QueryParser) parseFilter(key, value string) (Filter, error) {
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return Filter{}, fmt.Errorf("invalid filter format: %s", key)
	}

	column := parts[0]
	operator := FilterOperator(parts[1])

	// Valider l'opérateur
	validOperators := map[FilterOperator]bool{
		OpEqual: true, OpNotEqual: true, OpGreaterThan: true, OpGreaterEqual: true,
		OpLessThan: true, OpLessEqual: true, OpLike: true, OpILike: true,
		OpIn: true, OpIs: true,
	}

	if !validOperators[operator] {
		return Filter{}, fmt.Errorf("unknown operator: %s", operator)
	}

	return Filter{
		Column:   column,
		Operator: operator,
		Value:    value,
	}, nil
}

// parseSelect parse les colonnes à sélectionner
func (p *QueryParser) parseSelect(value string) []string {
	if value == "*" {
		return []string{"*"}
	}

	columns := strings.Split(value, ",")
	for i, col := range columns {
		columns[i] = strings.TrimSpace(col)
	}
	return columns
}

// parseOrder parse les clauses de tri
func (p *QueryParser) parseOrder(value string) ([]OrderClause, error) {
	clauses := strings.Split(value, ",")
	var orderClauses []OrderClause

	for _, clause := range clauses {
		parts := strings.Split(strings.TrimSpace(clause), ".")
		if len(parts) == 0 {
			continue
		}

		orderClause := OrderClause{
			Column:    parts[0],
			Direction: "asc", // défaut
			Nulls:     "",    // défaut
		}

		// Parser direction et nulls
		for i := 1; i < len(parts); i++ {
			switch parts[i] {
			case "asc", "desc":
				orderClause.Direction = parts[i]
			case "nullsfirst", "nullslast":
				orderClause.Nulls = strings.TrimPrefix(parts[i], "nulls")
			}
		}

		orderClauses = append(orderClauses, orderClause)
	}

	return orderClauses, nil
}

// isPostgRESTFilter vérifie si la clé est au format PostgREST (column=operator.value)
func (p *QueryParser) isPostgRESTFilter(key string) bool {
	// Vérifier si la valeur contient un opérateur PostgREST
	return !strings.Contains(key, ".") && key != "select" && key != "order" && key != "limit" && key != "offset" && key != "and" && key != "or"
}

// parsePostgRESTFilter parse un filtre au format PostgREST (column=operator.value)
func (p *QueryParser) parsePostgRESTFilter(key, value string) (Filter, error) {
	// Format: id=eq.1 ou name=like.john*
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return Filter{}, fmt.Errorf("invalid PostgREST filter format: %s", value)
	}

	operator := FilterOperator(parts[0])
	filterValue := strings.Join(parts[1:], ".")

	// Valider l'opérateur
	validOperators := map[FilterOperator]bool{
		OpEqual: true, OpNotEqual: true, OpGreaterThan: true, OpGreaterEqual: true,
		OpLessThan: true, OpLessEqual: true, OpLike: true, OpILike: true,
		OpIn: true, OpIs: true,
	}

	if !validOperators[operator] {
		return Filter{}, fmt.Errorf("unknown operator: %s", operator)
	}

	return Filter{
		Column:   key,
		Operator: operator,
		Value:    filterValue,
	}, nil
}

// parseLogicalCondition parse des conditions logiques complexes (and/or)
func (p *QueryParser) parseLogicalCondition(value string) (LogicalCondition, error) {
	// Format: (col1.op1.val1,col2.op2.val2)
	if !strings.HasPrefix(value, "(") || !strings.HasSuffix(value, ")") {
		return LogicalCondition{}, fmt.Errorf("logical condition must be wrapped in parentheses")
	}

	inner := strings.Trim(value, "()")
	conditions := []LogicalCondition{}
	filters := []Filter{}

	// Parser chaque condition séparée par des virgules
	parts := strings.Split(inner, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, ".") {
			filter, err := p.parseFilter(part, "")
			if err != nil {
				return LogicalCondition{}, err
			}
			// Pour les conditions logiques, la valeur est après l'opérateur
			filterParts := strings.Split(part, ".")
			if len(filterParts) >= 3 {
				filter.Column = filterParts[0]
				filter.Operator = FilterOperator(filterParts[1])
				filter.Value = strings.Join(filterParts[2:], ".")
			}
			filters = append(filters, filter)
		}
	}

	return LogicalCondition{
		Filters:    filters,
		Conditions: conditions,
		Negated:    false,
	}, nil
}
