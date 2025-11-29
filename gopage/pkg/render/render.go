package render

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/horos/gopage/pkg/engine"
)

// Component represents a renderable component from SQL results
type Component struct {
	Type       string                 // Component type: shell, text, table, list, card, form
	Properties map[string]interface{} // Component properties from SQL columns
}

// Renderer handles HTML template rendering
type Renderer struct {
	templates *template.Template
	funcs     template.FuncMap
}

// NewRenderer creates a new renderer with templates from the given directory
func NewRenderer(templateDir string) (*Renderer, error) {
	r := &Renderer{
		funcs: defaultFuncs(),
	}

	pattern := filepath.Join(templateDir, "**", "*.html")
	tmpl, err := template.New("").Funcs(r.funcs).ParseGlob(pattern)
	if err != nil {
		// Try flat structure
		pattern = filepath.Join(templateDir, "*.html")
		tmpl, err = template.New("").Funcs(r.funcs).ParseGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to parse templates: %w", err)
		}
	}

	// Also load component templates
	compPattern := filepath.Join(templateDir, "components", "*.html")
	tmpl, _ = tmpl.ParseGlob(compPattern)

	r.templates = tmpl
	return r, nil
}

// NewRendererFromFS creates a renderer from an embedded filesystem
func NewRendererFromFS(fsys embed.FS, root string) (*Renderer, error) {
	r := &Renderer{
		funcs: defaultFuncs(),
	}

	tmpl := template.New("").Funcs(r.funcs)

	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}

		name := strings.TrimPrefix(path, root+"/")
		name = strings.TrimSuffix(name, ".html")

		_, err = tmpl.New(name).Parse(string(content))
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse embedded templates: %w", err)
	}

	r.templates = tmpl
	return r, nil
}

// defaultFuncs returns the default template functions
func defaultFuncs() template.FuncMap {
	return template.FuncMap{
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"attr": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
		},
		"url": func(s string) template.URL {
			return template.URL(s)
		},
		"json": func(v interface{}) template.JS {
			return template.JS(fmt.Sprintf("%v", v))
		},
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title,
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
	}
}

// ParseComponents extracts components from SQL result rows
func (r *Renderer) ParseComponents(rows []engine.Row) []Component {
	var components []Component

	for _, row := range rows {
		comp := Component{
			Properties: make(map[string]interface{}),
		}

		// Extract component type
		if t, ok := row["component"].(string); ok {
			comp.Type = t
		} else {
			comp.Type = "text" // Default component
		}

		// Copy all properties
		for k, v := range row {
			if k != "component" {
				comp.Properties[k] = v
			}
		}

		components = append(components, comp)
	}

	return components
}

// RenderPage renders a full page with components
func (r *Renderer) RenderPage(components []Component, pageData map[string]interface{}) (string, error) {
	var buf bytes.Buffer

	// Prepare page data
	data := map[string]interface{}{
		"Components": components,
		"Page":       pageData,
	}

	// Find shell component for layout
	var shell *Component
	var content []Component
	for i, c := range components {
		if c.Type == "shell" {
			shell = &components[i]
		} else {
			content = append(content, c)
		}
	}

	// Render content components
	var contentHTML strings.Builder
	for _, c := range content {
		html, err := r.RenderComponent(c)
		if err != nil {
			return "", fmt.Errorf("failed to render %s component: %w", c.Type, err)
		}
		contentHTML.WriteString(html)
	}

	data["Content"] = template.HTML(contentHTML.String())

	// Use shell template or default layout
	layoutName := "layout"
	if shell != nil {
		for k, v := range shell.Properties {
			data[k] = v
		}
	}

	err := r.templates.ExecuteTemplate(&buf, layoutName, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute layout template: %w", err)
	}

	return buf.String(), nil
}

// RenderComponent renders a single component
func (r *Renderer) RenderComponent(comp Component) (string, error) {
	var buf bytes.Buffer

	templateName := "components/" + comp.Type
	if r.templates.Lookup(templateName) == nil {
		templateName = comp.Type
	}

	// Fallback to generic component template
	if r.templates.Lookup(templateName) == nil {
		return r.renderGenericComponent(comp)
	}

	err := r.templates.ExecuteTemplate(&buf, templateName, comp.Properties)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderGenericComponent renders a component without a specific template
func (r *Renderer) renderGenericComponent(comp Component) (string, error) {
	switch comp.Type {
	case "text":
		return r.renderText(comp.Properties)
	case "table":
		return r.renderTable(comp.Properties)
	case "list":
		return r.renderList(comp.Properties)
	case "card":
		return r.renderCard(comp.Properties)
	case "form":
		return r.renderForm(comp.Properties)
	case "debug":
		return r.renderDebug(comp.Properties)
	default:
		return fmt.Sprintf("<!-- Unknown component: %s -->", comp.Type), nil
	}
}

// renderText renders a simple text component
func (r *Renderer) renderText(props map[string]interface{}) (string, error) {
	var buf bytes.Buffer

	title, _ := props["title"].(string)
	contents, _ := props["contents"].(string)
	html, _ := props["html"].(string)

	buf.WriteString(`<div class="text-component">`)
	if title != "" {
		buf.WriteString(fmt.Sprintf(`<h2>%s</h2>`, template.HTMLEscapeString(title)))
	}
	if contents != "" {
		buf.WriteString(fmt.Sprintf(`<p>%s</p>`, template.HTMLEscapeString(contents)))
	}
	if html != "" {
		buf.WriteString(html) // Raw HTML (trusted)
	}
	buf.WriteString(`</div>`)

	return buf.String(), nil
}

// renderTable renders a table component
func (r *Renderer) renderTable(props map[string]interface{}) (string, error) {
	// Table rendering will use the actual data rows
	return `<div class="table-component"><!-- Table data will be rendered here --></div>`, nil
}

// renderList renders a list component
func (r *Renderer) renderList(props map[string]interface{}) (string, error) {
	return `<div class="list-component"><!-- List data will be rendered here --></div>`, nil
}

// renderCard renders a card component
func (r *Renderer) renderCard(props map[string]interface{}) (string, error) {
	var buf bytes.Buffer

	title, _ := props["title"].(string)
	description, _ := props["description"].(string)
	link, _ := props["link"].(string)

	buf.WriteString(`<div class="card">`)
	if title != "" {
		if link != "" {
			buf.WriteString(fmt.Sprintf(`<h3><a href="%s">%s</a></h3>`,
				template.HTMLEscapeString(link),
				template.HTMLEscapeString(title)))
		} else {
			buf.WriteString(fmt.Sprintf(`<h3>%s</h3>`, template.HTMLEscapeString(title)))
		}
	}
	if description != "" {
		buf.WriteString(fmt.Sprintf(`<p>%s</p>`, template.HTMLEscapeString(description)))
	}
	buf.WriteString(`</div>`)

	return buf.String(), nil
}

// renderForm renders a form component
func (r *Renderer) renderForm(props map[string]interface{}) (string, error) {
	return `<form class="form-component" method="POST"><!-- Form fields will be rendered here --></form>`, nil
}

// renderDebug renders debug output
func (r *Renderer) renderDebug(props map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(`<pre class="debug-component">`)
	for k, v := range props {
		buf.WriteString(fmt.Sprintf("%s: %v\n", k, v))
	}
	buf.WriteString(`</pre>`)
	return buf.String(), nil
}
