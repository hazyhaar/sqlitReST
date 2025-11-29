package server

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hazyhaar/sqlitrest/pkg/render"
)

// RequestParams holds parsed request parameters
type RequestParams struct {
	Query      map[string]string
	Form       map[string]string
	Path       map[string]string
	Page       int
	PerPage    int
	Sort       string
	SortDir    string
	Filters    map[string]string
	Search     string
	RequestURL *url.URL
}

// DefaultPerPage is the default number of items per page
const DefaultPerPage = 25

// ParseRequestParams extracts all parameters from an HTTP request
func ParseRequestParams(r *http.Request) *RequestParams {
	params := &RequestParams{
		Query:      make(map[string]string),
		Form:       make(map[string]string),
		Path:       make(map[string]string),
		Filters:    make(map[string]string),
		Page:       1,
		PerPage:    DefaultPerPage,
		SortDir:    "asc",
		RequestURL: r.URL,
	}

	// Parse query parameters
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params.Query[key] = values[0]
		}
	}

	// Parse form values for POST
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		if err := r.ParseForm(); err == nil {
			for key, values := range r.PostForm {
				if len(values) > 0 {
					params.Form[key] = values[0]
				}
			}
		}
	}

	// Extract pagination
	if page, ok := params.Query["page"]; ok {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params.Page = p
		}
	}

	if perPage, ok := params.Query["per_page"]; ok {
		if pp, err := strconv.Atoi(perPage); err == nil && pp > 0 && pp <= 100 {
			params.PerPage = pp
		}
	}

	// Also support limit/offset style
	if limit, ok := params.Query["limit"]; ok {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			params.PerPage = l
		}
	}

	if offset, ok := params.Query["offset"]; ok {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			params.Page = (o / params.PerPage) + 1
		}
	}

	// Extract sorting
	if sort, ok := params.Query["sort"]; ok {
		params.Sort = sort
	}

	if dir, ok := params.Query["dir"]; ok {
		if dir == "desc" || dir == "DESC" {
			params.SortDir = "desc"
		}
	}

	// Handle sort with direction suffix (e.g., "name.desc")
	if strings.Contains(params.Sort, ".") {
		parts := strings.SplitN(params.Sort, ".", 2)
		params.Sort = parts[0]
		if len(parts) > 1 && (parts[1] == "desc" || parts[1] == "DESC") {
			params.SortDir = "desc"
		}
	}

	// Extract search
	if search, ok := params.Query["q"]; ok {
		params.Search = search
	}
	if search, ok := params.Query["search"]; ok {
		params.Search = search
	}

	// Extract filters (anything that looks like column.op=value)
	for key, value := range params.Query {
		if strings.Contains(key, ".") {
			params.Filters[key] = value
		}
	}

	return params
}

// ToSQLParams converts RequestParams to a map for SQL binding
func (p *RequestParams) ToSQLParams() map[string]string {
	result := make(map[string]string)

	// Copy all query params
	for k, v := range p.Query {
		result[k] = v
	}

	// Copy all form params (override query)
	for k, v := range p.Form {
		result[k] = v
	}

	// Copy path params (override all)
	for k, v := range p.Path {
		result[k] = v
	}

	// Add pagination helpers
	result["_page"] = strconv.Itoa(p.Page)
	result["_per_page"] = strconv.Itoa(p.PerPage)
	result["_offset"] = strconv.Itoa((p.Page - 1) * p.PerPage)
	result["_limit"] = strconv.Itoa(p.PerPage)

	if p.Sort != "" {
		result["_sort"] = p.Sort
		result["_sort_dir"] = p.SortDir
		result["_order_by"] = p.Sort + " " + strings.ToUpper(p.SortDir)
	}

	if p.Search != "" {
		result["_search"] = p.Search
		result["_search_like"] = "%" + p.Search + "%"
	}

	return result
}

// BuildPagination creates pagination info for the given total count
func (p *RequestParams) BuildPagination(totalRows int) *render.PaginationInfo {
	totalPages := (totalRows + p.PerPage - 1) / p.PerPage
	if totalPages < 1 {
		totalPages = 1
	}

	pag := &render.PaginationInfo{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      totalRows,
		TotalPages: totalPages,
		HasPrev:    p.Page > 1,
		HasNext:    p.Page < totalPages,
		BaseURL:    p.RequestURL.Path,
	}

	// Build URLs
	baseQuery := p.RequestURL.Query()

	if pag.HasPrev {
		baseQuery.Set("page", strconv.Itoa(p.Page-1))
		pag.PrevURL = p.RequestURL.Path + "?" + baseQuery.Encode()
	}

	if pag.HasNext {
		baseQuery.Set("page", strconv.Itoa(p.Page+1))
		pag.NextURL = p.RequestURL.Path + "?" + baseQuery.Encode()
	}

	return pag
}

// ParseFilters parses filter parameters and returns SQL WHERE conditions
// Supports formats like: status.eq=active, price.gt=100, name.like=%john%
func ParseFilters(filters map[string]string) (conditions []string, values []interface{}) {
	for key, value := range filters {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			continue
		}

		column := parts[0]
		op := parts[1]

		// Validate column name (prevent injection)
		if !isValidColumnName(column) {
			continue
		}

		switch op {
		case "eq":
			conditions = append(conditions, column+" = ?")
			values = append(values, value)
		case "neq", "ne":
			conditions = append(conditions, column+" != ?")
			values = append(values, value)
		case "gt":
			conditions = append(conditions, column+" > ?")
			values = append(values, value)
		case "gte", "ge":
			conditions = append(conditions, column+" >= ?")
			values = append(values, value)
		case "lt":
			conditions = append(conditions, column+" < ?")
			values = append(values, value)
		case "lte", "le":
			conditions = append(conditions, column+" <= ?")
			values = append(values, value)
		case "like":
			conditions = append(conditions, column+" LIKE ?")
			values = append(values, value)
		case "ilike":
			conditions = append(conditions, column+" LIKE ? COLLATE NOCASE")
			values = append(values, value)
		case "in":
			// Split by comma
			vals := strings.Split(value, ",")
			placeholders := make([]string, len(vals))
			for i, v := range vals {
				placeholders[i] = "?"
				values = append(values, strings.TrimSpace(v))
			}
			conditions = append(conditions, column+" IN ("+strings.Join(placeholders, ",")+")")
		case "is":
			if strings.ToLower(value) == "null" {
				conditions = append(conditions, column+" IS NULL")
			} else if strings.ToLower(value) == "notnull" {
				conditions = append(conditions, column+" IS NOT NULL")
			}
		}
	}

	return
}

// isValidColumnName checks if a string is a valid column name
func isValidColumnName(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}

	for i, r := range s {
		if i == 0 {
			if !isLetter(r) && r != '_' {
				return false
			}
		} else {
			if !isLetter(r) && !isDigit(r) && r != '_' {
				return false
			}
		}
	}

	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
