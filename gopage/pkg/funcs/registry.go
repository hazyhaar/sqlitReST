package funcs

import (
	"context"
	"fmt"
	"sync"

	"zombiezen.com/go/sqlite"
)

// FuncType represents the type of a custom function
type FuncType int

const (
	FuncTypeScalar FuncType = iota // Returns a single value
	FuncTypeAggregate              // Aggregates multiple values
)

// FuncDef defines a custom SQL function
type FuncDef struct {
	Name          string
	NumArgs       int // -1 for variadic
	Type          FuncType
	Deterministic bool // Can be cached for same inputs
	Description   string
	ScalarFunc    ScalarFunc
	AggregateFunc AggregateFunc
}

// ScalarFunc is the signature for scalar functions
type ScalarFunc func(ctx context.Context, args []sqlite.Value) (interface{}, error)

// AggregateFunc is the signature for aggregate functions
type AggregateFunc interface {
	Step(ctx context.Context, args []sqlite.Value) error
	Final(ctx context.Context) (interface{}, error)
}

// Registry holds all registered custom functions
type Registry struct {
	mu    sync.RWMutex
	funcs map[string]*FuncDef

	// Configuration for external services
	LLMEndpoint string
	LLMAPIKey   string
	LLMModel    string

	HTTPTimeout int // seconds
}

// NewRegistry creates a new function registry
func NewRegistry() *Registry {
	r := &Registry{
		funcs:       make(map[string]*FuncDef),
		HTTPTimeout: 30,
		LLMModel:    "llama-3.3-70b", // Default Cerebras model
	}

	// Register built-in functions
	r.RegisterBuiltins()

	return r
}

// Register adds a function to the registry
func (r *Registry) Register(def *FuncDef) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if def.Name == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	if def.ScalarFunc == nil && def.AggregateFunc == nil {
		return fmt.Errorf("function %s must have either ScalarFunc or AggregateFunc", def.Name)
	}

	r.funcs[def.Name] = def
	return nil
}

// Get retrieves a function definition by name
func (r *Registry) Get(name string) (*FuncDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.funcs[name]
	return def, ok
}

// List returns all registered function names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.funcs))
	for name := range r.funcs {
		names = append(names, name)
	}
	return names
}

// ApplyToConnection registers all functions on a SQLite connection
func (r *Registry) ApplyToConnection(conn *sqlite.Conn) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, def := range r.funcs {
		if err := r.applyFunc(conn, def); err != nil {
			return fmt.Errorf("failed to register function %s: %w", def.Name, err)
		}
	}

	return nil
}

// applyFunc registers a single function on a connection
func (r *Registry) applyFunc(conn *sqlite.Conn, def *FuncDef) error {
	opts := &sqlite.FunctionImpl{
		NArgs:         def.NumArgs,
		Deterministic: def.Deterministic,
	}

	if def.ScalarFunc != nil {
		opts.Scalar = r.wrapScalar(def.ScalarFunc)
	}

	// Note: Aggregate functions require additional handling
	// that's more complex with zombiezen's API

	return conn.CreateFunction(def.Name, opts)
}

// wrapScalar wraps our ScalarFunc to match zombiezen's API
func (r *Registry) wrapScalar(fn ScalarFunc) func(ctx sqlite.Context, args []sqlite.Value) {
	return func(ctx sqlite.Context, args []sqlite.Value) {
		// Create a context with timeout
		goCtx := context.Background()

		result, err := fn(goCtx, args)
		if err != nil {
			ctx.ResultError(err)
			return
		}

		// Set result based on type
		switch v := result.(type) {
		case nil:
			ctx.ResultNull()
		case int:
			ctx.ResultInt(v)
		case int64:
			ctx.ResultInt64(v)
		case float64:
			ctx.ResultFloat(v)
		case string:
			ctx.ResultText(v)
		case []byte:
			ctx.ResultBlob(v)
		case bool:
			if v {
				ctx.ResultInt(1)
			} else {
				ctx.ResultInt(0)
			}
		default:
			ctx.ResultText(fmt.Sprintf("%v", v))
		}
	}
}

// RegisterBuiltins registers all built-in functions
func (r *Registry) RegisterBuiltins() {
	// String functions
	r.Register(&FuncDef{
		Name:          "gopage_version",
		NumArgs:       0,
		Deterministic: true,
		Description:   "Returns GoPage version",
		ScalarFunc:    funcVersion,
	})

	r.Register(&FuncDef{
		Name:          "uuid",
		NumArgs:       0,
		Deterministic: false,
		Description:   "Generates a UUID v4",
		ScalarFunc:    funcUUID,
	})

	r.Register(&FuncDef{
		Name:          "sha256",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Returns SHA256 hash of input",
		ScalarFunc:    funcSHA256,
	})

	r.Register(&FuncDef{
		Name:          "md5",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Returns MD5 hash of input",
		ScalarFunc:    funcMD5,
	})

	r.Register(&FuncDef{
		Name:          "base64_encode",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Encodes input to base64",
		ScalarFunc:    funcBase64Encode,
	})

	r.Register(&FuncDef{
		Name:          "base64_decode",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Decodes base64 input",
		ScalarFunc:    funcBase64Decode,
	})

	r.Register(&FuncDef{
		Name:          "url_encode",
		NumArgs:       1,
		Deterministic: true,
		Description:   "URL encodes input",
		ScalarFunc:    funcURLEncode,
	})

	r.Register(&FuncDef{
		Name:          "url_decode",
		NumArgs:       1,
		Deterministic: true,
		Description:   "URL decodes input",
		ScalarFunc:    funcURLDecode,
	})

	r.Register(&FuncDef{
		Name:          "json_extract_path",
		NumArgs:       2,
		Deterministic: true,
		Description:   "Extracts value from JSON using path",
		ScalarFunc:    funcJSONExtract,
	})

	r.Register(&FuncDef{
		Name:          "slugify",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Converts text to URL-friendly slug",
		ScalarFunc:    funcSlugify,
	})

	r.Register(&FuncDef{
		Name:          "truncate",
		NumArgs:       2,
		Deterministic: true,
		Description:   "Truncates text to specified length with ellipsis",
		ScalarFunc:    funcTruncate,
	})

	r.Register(&FuncDef{
		Name:          "strip_html",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Removes HTML tags from text",
		ScalarFunc:    funcStripHTML,
	})

	r.Register(&FuncDef{
		Name:          "markdown_to_html",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Converts markdown to HTML (basic)",
		ScalarFunc:    funcMarkdownToHTML,
	})

	// Date functions
	r.Register(&FuncDef{
		Name:          "now_utc",
		NumArgs:       0,
		Deterministic: false,
		Description:   "Returns current UTC timestamp",
		ScalarFunc:    funcNowUTC,
	})

	r.Register(&FuncDef{
		Name:          "format_date",
		NumArgs:       2,
		Deterministic: true,
		Description:   "Formats date according to layout",
		ScalarFunc:    funcFormatDate,
	})

	r.Register(&FuncDef{
		Name:          "time_ago",
		NumArgs:       1,
		Deterministic: false,
		Description:   "Returns human-readable time ago string",
		ScalarFunc:    funcTimeAgo,
	})

	// Utility functions
	r.Register(&FuncDef{
		Name:          "coalesce_empty",
		NumArgs:       -1, // Variadic
		Deterministic: true,
		Description:   "Returns first non-empty string argument",
		ScalarFunc:    funcCoalesceEmpty,
	})

	r.Register(&FuncDef{
		Name:          "format_number",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Formats number with thousands separator",
		ScalarFunc:    funcFormatNumber,
	})

	r.Register(&FuncDef{
		Name:          "format_bytes",
		NumArgs:       1,
		Deterministic: true,
		Description:   "Formats bytes as human readable (KB, MB, etc)",
		ScalarFunc:    funcFormatBytes,
	})
}
