package render

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/horos/gopage/pkg/engine"
)

// FormField represents a form field definition from SQL
type FormField struct {
	Name        string
	Type        string // text, email, password, number, textarea, select, checkbox, radio, hidden, file, date, time, datetime
	Label       string
	Value       string
	Placeholder string
	Required    bool
	Readonly    bool
	Disabled    bool
	Pattern     string
	Min         string
	Max         string
	Step        string
	Options     []SelectOption // For select, radio
	Help        string
	Width       string // full, half, third
}

// SelectOption represents an option in a select or radio field
type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

// FormComponent represents a complete form definition
type FormComponent struct {
	ID          string
	Method      string // GET, POST
	Action      string
	Title       string
	Description string
	Fields      []FormField
	SubmitLabel string
	CancelURL   string
	Reset       bool

	// HTMX attributes
	HxPost    string
	HxPut     string
	HxDelete  string
	HxTarget  string
	HxSwap    string
	HxConfirm string
}

// ParseFormComponent parses form definition and fields from SQL rows
func (r *Renderer) ParseFormComponent(rows []engine.Row) *FormComponent {
	form := &FormComponent{
		Method:      "POST",
		SubmitLabel: "Envoyer",
	}

	for _, row := range rows {
		// Check if this is the form definition row
		if compType, ok := row["component"].(string); ok && compType == "form" {
			r.parseFormDefinition(form, row)
			continue
		}

		// Otherwise, it's a field definition
		field := r.parseFieldDefinition(row)
		if field.Name != "" {
			form.Fields = append(form.Fields, field)
		}
	}

	return form
}

// parseFormDefinition extracts form properties from a row
func (r *Renderer) parseFormDefinition(form *FormComponent, row engine.Row) {
	if id, ok := row["id"].(string); ok {
		form.ID = id
	}
	if method, ok := row["method"].(string); ok {
		form.Method = strings.ToUpper(method)
	}
	if action, ok := row["action"].(string); ok {
		form.Action = action
	}
	if title, ok := row["title"].(string); ok {
		form.Title = title
	}
	if desc, ok := row["description"].(string); ok {
		form.Description = desc
	}
	if submit, ok := row["submit_label"].(string); ok {
		form.SubmitLabel = submit
	}
	if cancel, ok := row["cancel_url"].(string); ok {
		form.CancelURL = cancel
	}
	if reset, ok := row["reset"].(bool); ok {
		form.Reset = reset
	}
	if reset, ok := row["reset"].(int64); ok && reset == 1 {
		form.Reset = true
	}

	// HTMX attributes
	if hxPost, ok := row["hx_post"].(string); ok {
		form.HxPost = hxPost
	}
	if hxPut, ok := row["hx_put"].(string); ok {
		form.HxPut = hxPut
	}
	if hxDelete, ok := row["hx_delete"].(string); ok {
		form.HxDelete = hxDelete
	}
	if hxTarget, ok := row["hx_target"].(string); ok {
		form.HxTarget = hxTarget
	}
	if hxSwap, ok := row["hx_swap"].(string); ok {
		form.HxSwap = hxSwap
	}
	if hxConfirm, ok := row["hx_confirm"].(string); ok {
		form.HxConfirm = hxConfirm
	}
}

// parseFieldDefinition extracts field properties from a row
func (r *Renderer) parseFieldDefinition(row engine.Row) FormField {
	field := FormField{
		Type: "text",
	}

	if name, ok := row["name"].(string); ok {
		field.Name = name
	}
	if fieldType, ok := row["type"].(string); ok {
		field.Type = fieldType
	}
	if label, ok := row["label"].(string); ok {
		field.Label = label
	} else {
		field.Label = formatColumnLabel(field.Name)
	}
	if value, ok := row["value"].(string); ok {
		field.Value = value
	}
	if placeholder, ok := row["placeholder"].(string); ok {
		field.Placeholder = placeholder
	}
	if required, ok := row["required"].(bool); ok {
		field.Required = required
	}
	if required, ok := row["required"].(int64); ok && required == 1 {
		field.Required = true
	}
	if readonly, ok := row["readonly"].(bool); ok {
		field.Readonly = readonly
	}
	if disabled, ok := row["disabled"].(bool); ok {
		field.Disabled = disabled
	}
	if pattern, ok := row["pattern"].(string); ok {
		field.Pattern = pattern
	}
	if min, ok := row["min"].(string); ok {
		field.Min = min
	}
	if max, ok := row["max"].(string); ok {
		field.Max = max
	}
	if step, ok := row["step"].(string); ok {
		field.Step = step
	}
	if help, ok := row["help"].(string); ok {
		field.Help = help
	}
	if width, ok := row["width"].(string); ok {
		field.Width = width
	}

	// Parse options for select/radio (format: "value1:Label1,value2:Label2")
	if options, ok := row["options"].(string); ok {
		field.Options = parseSelectOptions(options)
	}

	// Check if current value should be selected
	for i := range field.Options {
		if field.Options[i].Value == field.Value {
			field.Options[i].Selected = true
		}
	}

	return field
}

// parseSelectOptions parses options string into SelectOption slice
func parseSelectOptions(s string) []SelectOption {
	var options []SelectOption

	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		opt := SelectOption{}
		if idx := strings.Index(part, ":"); idx > 0 {
			opt.Value = strings.TrimSpace(part[:idx])
			opt.Label = strings.TrimSpace(part[idx+1:])
		} else {
			opt.Value = part
			opt.Label = part
		}

		options = append(options, opt)
	}

	return options
}

// RenderDynamicForm renders a complete form from FormComponent
func (r *Renderer) RenderDynamicForm(form *FormComponent) (string, error) {
	var buf bytes.Buffer

	// Form open tag
	buf.WriteString(`<form class="form-component fade-in"`)

	if form.ID != "" {
		buf.WriteString(fmt.Sprintf(` id="%s"`, template.HTMLEscapeString(form.ID)))
	}

	buf.WriteString(fmt.Sprintf(` method="%s"`, form.Method))

	if form.Action != "" {
		buf.WriteString(fmt.Sprintf(` action="%s"`, template.HTMLEscapeString(form.Action)))
	}

	// HTMX attributes
	if form.HxPost != "" {
		buf.WriteString(fmt.Sprintf(` hx-post="%s"`, template.HTMLEscapeString(form.HxPost)))
	}
	if form.HxPut != "" {
		buf.WriteString(fmt.Sprintf(` hx-put="%s"`, template.HTMLEscapeString(form.HxPut)))
	}
	if form.HxDelete != "" {
		buf.WriteString(fmt.Sprintf(` hx-delete="%s"`, template.HTMLEscapeString(form.HxDelete)))
	}
	if form.HxTarget != "" {
		buf.WriteString(fmt.Sprintf(` hx-target="%s"`, template.HTMLEscapeString(form.HxTarget)))
	}
	if form.HxSwap != "" {
		buf.WriteString(fmt.Sprintf(` hx-swap="%s"`, template.HTMLEscapeString(form.HxSwap)))
	}
	if form.HxConfirm != "" {
		buf.WriteString(fmt.Sprintf(` hx-confirm="%s"`, template.HTMLEscapeString(form.HxConfirm)))
	}

	buf.WriteString(` x-data="{ submitting: false }" @submit="submitting = true">`)

	// Title and description
	if form.Title != "" {
		buf.WriteString(fmt.Sprintf(`<h3>%s</h3>`, template.HTMLEscapeString(form.Title)))
	}
	if form.Description != "" {
		buf.WriteString(fmt.Sprintf(`<p>%s</p>`, template.HTMLEscapeString(form.Description)))
	}

	// Fields
	buf.WriteString(`<div class="form-fields">`)
	for _, field := range form.Fields {
		buf.WriteString(r.renderFormField(field))
	}
	buf.WriteString(`</div>`)

	// Footer with buttons
	buf.WriteString(`<footer class="form-actions">`)

	// Submit button
	buf.WriteString(`<button type="submit" :disabled="submitting" :aria-busy="submitting">`)
	buf.WriteString(`<span x-show="!submitting">`)
	buf.WriteString(template.HTMLEscapeString(form.SubmitLabel))
	buf.WriteString(`</span>`)
	buf.WriteString(`<span x-show="submitting">Envoi...</span>`)
	buf.WriteString(`</button>`)

	// Reset button
	if form.Reset {
		buf.WriteString(` <button type="reset" class="secondary outline">Réinitialiser</button>`)
	}

	// Cancel link
	if form.CancelURL != "" {
		buf.WriteString(fmt.Sprintf(` <a href="%s" role="button" class="outline">Annuler</a>`,
			template.HTMLEscapeString(form.CancelURL)))
	}

	buf.WriteString(`</footer>`)
	buf.WriteString(`</form>`)

	return buf.String(), nil
}

// renderFormField renders a single form field
func (r *Renderer) renderFormField(field FormField) string {
	var buf bytes.Buffer

	// Field wrapper
	widthClass := ""
	if field.Width != "" {
		widthClass = fmt.Sprintf(" field-%s", field.Width)
	}
	buf.WriteString(fmt.Sprintf(`<div class="form-field%s">`, widthClass))

	switch field.Type {
	case "hidden":
		buf.WriteString(r.renderHiddenField(field))
	case "textarea":
		buf.WriteString(r.renderTextareaField(field))
	case "select":
		buf.WriteString(r.renderSelectField(field))
	case "checkbox":
		buf.WriteString(r.renderCheckboxField(field))
	case "radio":
		buf.WriteString(r.renderRadioField(field))
	default:
		buf.WriteString(r.renderInputField(field))
	}

	buf.WriteString(`</div>`)
	return buf.String()
}

func (r *Renderer) renderInputField(field FormField) string {
	var buf bytes.Buffer

	// Label
	buf.WriteString(fmt.Sprintf(`<label for="%s">%s`, field.Name, template.HTMLEscapeString(field.Label)))
	if field.Required {
		buf.WriteString(`<span class="required">*</span>`)
	}
	buf.WriteString(`</label>`)

	// Input
	buf.WriteString(fmt.Sprintf(`<input type="%s" id="%s" name="%s"`,
		field.Type, field.Name, field.Name))

	if field.Value != "" {
		buf.WriteString(fmt.Sprintf(` value="%s"`, template.HTMLEscapeString(field.Value)))
	}
	if field.Placeholder != "" {
		buf.WriteString(fmt.Sprintf(` placeholder="%s"`, template.HTMLEscapeString(field.Placeholder)))
	}
	if field.Required {
		buf.WriteString(` required`)
	}
	if field.Readonly {
		buf.WriteString(` readonly`)
	}
	if field.Disabled {
		buf.WriteString(` disabled`)
	}
	if field.Pattern != "" {
		buf.WriteString(fmt.Sprintf(` pattern="%s"`, template.HTMLEscapeString(field.Pattern)))
	}
	if field.Min != "" {
		buf.WriteString(fmt.Sprintf(` min="%s"`, field.Min))
	}
	if field.Max != "" {
		buf.WriteString(fmt.Sprintf(` max="%s"`, field.Max))
	}
	if field.Step != "" {
		buf.WriteString(fmt.Sprintf(` step="%s"`, field.Step))
	}

	buf.WriteString(`>`)

	// Help text
	if field.Help != "" {
		buf.WriteString(fmt.Sprintf(`<small>%s</small>`, template.HTMLEscapeString(field.Help)))
	}

	return buf.String()
}

func (r *Renderer) renderTextareaField(field FormField) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf(`<label for="%s">%s`, field.Name, template.HTMLEscapeString(field.Label)))
	if field.Required {
		buf.WriteString(`<span class="required">*</span>`)
	}
	buf.WriteString(`</label>`)

	buf.WriteString(fmt.Sprintf(`<textarea id="%s" name="%s"`, field.Name, field.Name))

	if field.Placeholder != "" {
		buf.WriteString(fmt.Sprintf(` placeholder="%s"`, template.HTMLEscapeString(field.Placeholder)))
	}
	if field.Required {
		buf.WriteString(` required`)
	}
	if field.Readonly {
		buf.WriteString(` readonly`)
	}
	if field.Disabled {
		buf.WriteString(` disabled`)
	}

	buf.WriteString(`>`)
	buf.WriteString(template.HTMLEscapeString(field.Value))
	buf.WriteString(`</textarea>`)

	if field.Help != "" {
		buf.WriteString(fmt.Sprintf(`<small>%s</small>`, template.HTMLEscapeString(field.Help)))
	}

	return buf.String()
}

func (r *Renderer) renderSelectField(field FormField) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf(`<label for="%s">%s`, field.Name, template.HTMLEscapeString(field.Label)))
	if field.Required {
		buf.WriteString(`<span class="required">*</span>`)
	}
	buf.WriteString(`</label>`)

	buf.WriteString(fmt.Sprintf(`<select id="%s" name="%s"`, field.Name, field.Name))
	if field.Required {
		buf.WriteString(` required`)
	}
	if field.Disabled {
		buf.WriteString(` disabled`)
	}
	buf.WriteString(`>`)

	// Placeholder option
	if field.Placeholder != "" {
		buf.WriteString(fmt.Sprintf(`<option value="">%s</option>`, template.HTMLEscapeString(field.Placeholder)))
	}

	// Options
	for _, opt := range field.Options {
		selected := ""
		if opt.Selected {
			selected = " selected"
		}
		buf.WriteString(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
			template.HTMLEscapeString(opt.Value),
			selected,
			template.HTMLEscapeString(opt.Label)))
	}

	buf.WriteString(`</select>`)

	if field.Help != "" {
		buf.WriteString(fmt.Sprintf(`<small>%s</small>`, template.HTMLEscapeString(field.Help)))
	}

	return buf.String()
}

func (r *Renderer) renderCheckboxField(field FormField) string {
	var buf bytes.Buffer

	checked := ""
	if field.Value == "1" || field.Value == "true" {
		checked = " checked"
	}

	buf.WriteString(`<label>`)
	buf.WriteString(fmt.Sprintf(`<input type="checkbox" id="%s" name="%s" value="1"%s`,
		field.Name, field.Name, checked))

	if field.Required {
		buf.WriteString(` required`)
	}
	if field.Disabled {
		buf.WriteString(` disabled`)
	}
	buf.WriteString(`>`)

	buf.WriteString(template.HTMLEscapeString(field.Label))
	buf.WriteString(`</label>`)

	if field.Help != "" {
		buf.WriteString(fmt.Sprintf(`<small>%s</small>`, template.HTMLEscapeString(field.Help)))
	}

	return buf.String()
}

func (r *Renderer) renderRadioField(field FormField) string {
	var buf bytes.Buffer

	buf.WriteString(`<fieldset>`)
	buf.WriteString(fmt.Sprintf(`<legend>%s`, template.HTMLEscapeString(field.Label)))
	if field.Required {
		buf.WriteString(`<span class="required">*</span>`)
	}
	buf.WriteString(`</legend>`)

	for _, opt := range field.Options {
		checked := ""
		if opt.Selected || opt.Value == field.Value {
			checked = " checked"
		}

		buf.WriteString(`<label>`)
		buf.WriteString(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s`,
			field.Name, template.HTMLEscapeString(opt.Value), checked))

		if field.Required {
			buf.WriteString(` required`)
		}
		if field.Disabled {
			buf.WriteString(` disabled`)
		}
		buf.WriteString(`>`)

		buf.WriteString(template.HTMLEscapeString(opt.Label))
		buf.WriteString(`</label>`)
	}

	buf.WriteString(`</fieldset>`)

	if field.Help != "" {
		buf.WriteString(fmt.Sprintf(`<small>%s</small>`, template.HTMLEscapeString(field.Help)))
	}

	return buf.String()
}

func (r *Renderer) renderHiddenField(field FormField) string {
	return fmt.Sprintf(`<input type="hidden" id="%s" name="%s" value="%s">`,
		field.Name, field.Name, template.HTMLEscapeString(field.Value))
}
