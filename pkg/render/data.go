package render

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/hazyhaar/sqlitrest/pkg/engine"
)

// DataComponent represents a component that displays data from SQL queries
type DataComponent struct {
	Type       string                 // table, list, cards
	Title      string                 // Optional title
	Columns    []ColumnDef            // Column definitions
	Rows       []engine.Row           // Data rows
	Pagination *PaginationInfo        // Pagination info
	Options    map[string]interface{} // Additional options
}

// ColumnDef defines a table column
type ColumnDef struct {
	Name      string // Column name from SQL
	Label     string // Display label
	Type      string // text, number, date, link, html, badge
	Sortable  bool
	Align     string // left, center, right
	Width     string // CSS width
	LinkField string // For link type: field containing the URL
	Format    string // Format string for dates/numbers
}

// PaginationInfo holds pagination state
type PaginationInfo struct {
	Page       int
	PerPage    int
	Total      int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevURL    string
	NextURL    string
	BaseURL    string
}

// ParseDataComponents groups SQL rows into data components
// This handles the SQLPage pattern where a component definition row is followed by data rows
func (r *Renderer) ParseDataComponents(rows []engine.Row) []Component {
	var components []Component
	var currentDataComp *DataComponent
	var pendingRows []engine.Row

	for _, row := range rows {
		compType, hasComponent := row["component"].(string)

		if hasComponent {
			// Flush pending data rows to previous component
			if currentDataComp != nil && len(pendingRows) > 0 {
				currentDataComp.Rows = pendingRows
				components = append(components, r.dataComponentToComponent(currentDataComp))
				pendingRows = nil
			}

			// Check if this is a data component
			if isDataComponent(compType) {
				currentDataComp = r.parseDataComponentDef(compType, row)
			} else {
				currentDataComp = nil
				// Regular component
				comp := Component{
					Type:       compType,
					Properties: make(map[string]interface{}),
				}
				for k, v := range row {
					if k != "component" {
						comp.Properties[k] = v
					}
				}
				components = append(components, comp)
			}
		} else if currentDataComp != nil {
			// Data row for current component
			pendingRows = append(pendingRows, row)
		}
	}

	// Flush final data component
	if currentDataComp != nil && len(pendingRows) > 0 {
		currentDataComp.Rows = pendingRows
		components = append(components, r.dataComponentToComponent(currentDataComp))
	}

	return components
}

// isDataComponent checks if a component type expects data rows
func isDataComponent(compType string) bool {
	switch compType {
	case "table", "list", "cards", "datagrid":
		return true
	default:
		return false
	}
}

// parseDataComponentDef parses a data component definition row
func (r *Renderer) parseDataComponentDef(compType string, row engine.Row) *DataComponent {
	dc := &DataComponent{
		Type:    compType,
		Options: make(map[string]interface{}),
	}

	// Extract common properties
	if title, ok := row["title"].(string); ok {
		dc.Title = title
	}

	// Extract columns definition if present
	if cols, ok := row["columns"].(string); ok {
		dc.Columns = parseColumnDefs(cols)
	}

	// Copy remaining options
	for k, v := range row {
		if k != "component" && k != "title" && k != "columns" {
			dc.Options[k] = v
		}
	}

	return dc
}

// parseColumnDefs parses column definitions from a string
// Format: "id:ID:number, name:Nom:text:sortable, email:Email:link"
func parseColumnDefs(s string) []ColumnDef {
	var cols []ColumnDef

	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		fields := strings.Split(part, ":")
		col := ColumnDef{
			Name:  strings.TrimSpace(fields[0]),
			Label: strings.TrimSpace(fields[0]), // Default to name
			Type:  "text",
		}

		if len(fields) > 1 {
			col.Label = strings.TrimSpace(fields[1])
		}
		if len(fields) > 2 {
			col.Type = strings.TrimSpace(fields[2])
		}
		if len(fields) > 3 {
			for _, opt := range fields[3:] {
				switch strings.TrimSpace(opt) {
				case "sortable":
					col.Sortable = true
				case "left", "center", "right":
					col.Align = opt
				}
			}
		}

		cols = append(cols, col)
	}

	return cols
}

// dataComponentToComponent converts a DataComponent to a renderable Component
func (r *Renderer) dataComponentToComponent(dc *DataComponent) Component {
	props := make(map[string]interface{})
	props["title"] = dc.Title
	props["columns"] = dc.Columns
	props["rows"] = dc.Rows
	props["pagination"] = dc.Pagination

	// Auto-detect columns from first row if not specified
	if len(dc.Columns) == 0 && len(dc.Rows) > 0 {
		props["columns"] = autoDetectColumns(dc.Rows[0])
	}

	// Copy options
	for k, v := range dc.Options {
		props[k] = v
	}

	return Component{
		Type:       dc.Type,
		Properties: props,
	}
}

// autoDetectColumns creates column definitions from row keys
func autoDetectColumns(row engine.Row) []ColumnDef {
	var cols []ColumnDef
	var keys []string

	// Collect and sort keys for consistent ordering
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		cols = append(cols, ColumnDef{
			Name:  k,
			Label: formatColumnLabel(k),
			Type:  detectColumnType(row[k]),
		})
	}

	return cols
}

// formatColumnLabel converts snake_case to Title Case
func formatColumnLabel(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	return strings.Title(s)
}

// detectColumnType guesses the column type from a value
func detectColumnType(v interface{}) string {
	switch v.(type) {
	case int, int64, float64:
		return "number"
	case bool:
		return "boolean"
	default:
		s := fmt.Sprintf("%v", v)
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			return "link"
		}
		return "text"
	}
}

// RenderDataTable renders a table with data
func (r *Renderer) RenderDataTable(dc DataComponent) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<div class="table-component fade-in">`)

	if dc.Title != "" {
		buf.WriteString(fmt.Sprintf(`<h3>%s</h3>`, template.HTMLEscapeString(dc.Title)))
	}

	buf.WriteString(`<figure><table role="grid">`)

	// Render header
	buf.WriteString(`<thead><tr>`)
	cols := dc.Columns
	if len(cols) == 0 && len(dc.Rows) > 0 {
		cols = autoDetectColumns(dc.Rows[0])
	}
	for _, col := range cols {
		sortAttr := ""
		if col.Sortable {
			sortAttr = ` class="sortable" hx-get="?sort=` + col.Name + `" hx-target="closest table tbody" hx-swap="innerHTML"`
		}
		buf.WriteString(fmt.Sprintf(`<th scope="col"%s>%s</th>`, sortAttr, template.HTMLEscapeString(col.Label)))
	}
	buf.WriteString(`</tr></thead>`)

	// Render body
	buf.WriteString(`<tbody>`)
	if len(dc.Rows) == 0 {
		buf.WriteString(fmt.Sprintf(`<tr><td colspan="%d">Aucune donnée</td></tr>`, len(cols)))
	} else {
		for _, row := range dc.Rows {
			buf.WriteString(`<tr>`)
			for _, col := range cols {
				val := row[col.Name]
				buf.WriteString(`<td>`)
				buf.WriteString(r.renderCellValue(val, col))
				buf.WriteString(`</td>`)
			}
			buf.WriteString(`</tr>`)
		}
	}
	buf.WriteString(`</tbody>`)

	buf.WriteString(`</table></figure>`)

	// Render pagination
	if dc.Pagination != nil && dc.Pagination.TotalPages > 1 {
		buf.WriteString(r.renderPagination(dc.Pagination))
	}

	buf.WriteString(`</div>`)

	return buf.String(), nil
}

// renderCellValue renders a single cell value based on column type
func (r *Renderer) renderCellValue(val interface{}, col ColumnDef) string {
	if val == nil {
		return `<span class="null">-</span>`
	}

	s := fmt.Sprintf("%v", val)

	switch col.Type {
	case "link":
		return fmt.Sprintf(`<a href="%s">%s</a>`,
			template.HTMLEscapeString(s),
			template.HTMLEscapeString(s))
	case "html":
		return s // Raw HTML
	case "badge":
		return fmt.Sprintf(`<mark>%s</mark>`, template.HTMLEscapeString(s))
	case "number":
		return fmt.Sprintf(`<span class="number">%s</span>`, s)
	case "boolean":
		if s == "true" || s == "1" {
			return `<span class="badge-success">Oui</span>`
		}
		return `<span class="badge-error">Non</span>`
	default:
		return template.HTMLEscapeString(s)
	}
}

// RenderDataList renders a list of items
func (r *Renderer) RenderDataList(dc DataComponent) (string, error) {
	var buf bytes.Buffer

	listType := "ul"
	if ordered, ok := dc.Options["ordered"].(bool); ok && ordered {
		listType = "ol"
	}

	buf.WriteString(`<div class="list-component fade-in">`)

	if dc.Title != "" {
		buf.WriteString(fmt.Sprintf(`<h3>%s</h3>`, template.HTMLEscapeString(dc.Title)))
	}

	buf.WriteString(fmt.Sprintf(`<%s>`, listType))

	for _, row := range dc.Rows {
		buf.WriteString(`<li>`)

		// Check for specific display fields
		if title, ok := row["title"].(string); ok {
			if link, ok := row["link"].(string); ok {
				buf.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`,
					template.HTMLEscapeString(link),
					template.HTMLEscapeString(title)))
			} else {
				buf.WriteString(template.HTMLEscapeString(title))
			}
			if desc, ok := row["description"].(string); ok {
				buf.WriteString(fmt.Sprintf(` <small>%s</small>`, template.HTMLEscapeString(desc)))
			}
		} else {
			// Fallback: display first text value
			for _, v := range row {
				buf.WriteString(fmt.Sprintf("%v", v))
				break
			}
		}

		buf.WriteString(`</li>`)
	}

	buf.WriteString(fmt.Sprintf(`</%s>`, listType))
	buf.WriteString(`</div>`)

	return buf.String(), nil
}

// RenderDataCards renders a grid of cards
func (r *Renderer) RenderDataCards(dc DataComponent) (string, error) {
	var buf bytes.Buffer

	buf.WriteString(`<div class="cards-component fade-in" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 1rem;">`)

	if dc.Title != "" {
		buf.WriteString(fmt.Sprintf(`<h3 style="grid-column: 1 / -1;">%s</h3>`, template.HTMLEscapeString(dc.Title)))
	}

	for _, row := range dc.Rows {
		buf.WriteString(`<article class="card">`)

		if img, ok := row["image"].(string); ok {
			buf.WriteString(fmt.Sprintf(`<header><img src="%s" alt="" loading="lazy"></header>`,
				template.HTMLEscapeString(img)))
		}

		if title, ok := row["title"].(string); ok {
			if link, ok := row["link"].(string); ok {
				buf.WriteString(fmt.Sprintf(`<h4><a href="%s">%s</a></h4>`,
					template.HTMLEscapeString(link),
					template.HTMLEscapeString(title)))
			} else {
				buf.WriteString(fmt.Sprintf(`<h4>%s</h4>`, template.HTMLEscapeString(title)))
			}
		}

		if desc, ok := row["description"].(string); ok {
			buf.WriteString(fmt.Sprintf(`<p>%s</p>`, template.HTMLEscapeString(desc)))
		}

		if footer, ok := row["footer"].(string); ok {
			buf.WriteString(fmt.Sprintf(`<footer><small>%s</small></footer>`,
				template.HTMLEscapeString(footer)))
		}

		buf.WriteString(`</article>`)
	}

	buf.WriteString(`</div>`)

	return buf.String(), nil
}

// renderPagination renders pagination controls
func (r *Renderer) renderPagination(p *PaginationInfo) string {
	var buf bytes.Buffer

	buf.WriteString(`<nav class="pagination">`)
	buf.WriteString(`<ul>`)

	// Previous
	if p.HasPrev {
		buf.WriteString(fmt.Sprintf(
			`<li><a href="%s" hx-get="%s" hx-target="closest .table-component" hx-swap="outerHTML">&laquo; Précédent</a></li>`,
			p.PrevURL, p.PrevURL))
	} else {
		buf.WriteString(`<li><a href="#" aria-disabled="true">&laquo; Précédent</a></li>`)
	}

	// Page info
	buf.WriteString(fmt.Sprintf(`<li><span>Page %d / %d</span></li>`, p.Page, p.TotalPages))

	// Next
	if p.HasNext {
		buf.WriteString(fmt.Sprintf(
			`<li><a href="%s" hx-get="%s" hx-target="closest .table-component" hx-swap="outerHTML">Suivant &raquo;</a></li>`,
			p.NextURL, p.NextURL))
	} else {
		buf.WriteString(`<li><a href="#" aria-disabled="true">Suivant &raquo;</a></li>`)
	}

	buf.WriteString(`</ul>`)
	buf.WriteString(`</nav>`)

	return buf.String()
}
