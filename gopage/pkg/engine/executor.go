package engine

import (
	"context"
	"fmt"
	"sync"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/horos/gopage/pkg/funcs"
)

// Row represents a single result row as key-value pairs
type Row map[string]interface{}

// Result holds query execution results
type Result struct {
	Rows     []Row
	Columns  []string
	Affected int64
}

// SQLExecutor executes SQL queries using zombiezen
type SQLExecutor struct {
	pool     *sqlitex.Pool
	registry *funcs.Registry

	// Track connections that have had functions applied
	appliedConns sync.Map
}

// NewSQLExecutor creates a new SQL executor
func NewSQLExecutor(dbPath string) (*SQLExecutor, error) {
	pool, err := sqlitex.NewPool(dbPath, sqlitex.PoolOptions{
		PoolSize: 10,
		Flags:    sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	return &SQLExecutor{pool: pool}, nil
}

// SetRegistry sets the function registry for custom SQL functions
func (e *SQLExecutor) SetRegistry(registry *funcs.Registry) {
	e.registry = registry
}

// GetRegistry returns the function registry
func (e *SQLExecutor) GetRegistry() *funcs.Registry {
	return e.registry
}

// applyFunctionsToConn applies registered functions to a connection if not already done
func (e *SQLExecutor) applyFunctionsToConn(conn *sqlite.Conn) error {
	if e.registry == nil {
		return nil
	}

	// Use pointer as key to track which connections have been setup
	ptr := fmt.Sprintf("%p", conn)
	if _, loaded := e.appliedConns.LoadOrStore(ptr, true); loaded {
		// Already applied
		return nil
	}

	return e.registry.ApplyToConnection(conn)
}

// Close closes the connection pool
func (e *SQLExecutor) Close() error {
	return e.pool.Close()
}

// Execute runs a SQL statement and returns results
func (e *SQLExecutor) Execute(ctx context.Context, sql string) (*Result, error) {
	conn, err := e.pool.Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer e.pool.Put(conn)

	// Apply custom functions to connection
	if err := e.applyFunctionsToConn(conn); err != nil {
		return nil, fmt.Errorf("failed to apply functions: %w", err)
	}

	return e.executeOnConn(conn, sql)
}

// executeOnConn executes SQL on a specific connection
func (e *SQLExecutor) executeOnConn(conn *sqlite.Conn, sql string) (*Result, error) {
	stmt, err := conn.Prepare(sql)
	if err != nil {
		return nil, fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Finalize()

	result := &Result{
		Rows:    make([]Row, 0),
		Columns: make([]string, 0),
	}

	// Get column names
	colCount := stmt.ColumnCount()
	for i := 0; i < colCount; i++ {
		result.Columns = append(result.Columns, stmt.ColumnName(i))
	}

	// Read rows
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("step failed: %w", err)
		}
		if !hasRow {
			break
		}

		row := make(Row)
		for i := 0; i < colCount; i++ {
			colName := result.Columns[i]
			row[colName] = e.readColumnValue(stmt, i)
		}
		result.Rows = append(result.Rows, row)
	}

	// For non-SELECT statements, get affected rows
	if len(result.Columns) == 0 {
		result.Affected = int64(conn.Changes())
	}

	return result, nil
}

// readColumnValue reads a value from a column based on its type
func (e *SQLExecutor) readColumnValue(stmt *sqlite.Stmt, idx int) interface{} {
	switch stmt.ColumnType(idx) {
	case sqlite.TypeNull:
		return nil
	case sqlite.TypeInteger:
		return stmt.ColumnInt64(idx)
	case sqlite.TypeFloat:
		return stmt.ColumnFloat(idx)
	case sqlite.TypeText:
		return stmt.ColumnText(idx)
	case sqlite.TypeBlob:
		return stmt.ColumnBytesUnsafe(idx)
	default:
		return stmt.ColumnText(idx)
	}
}

// ExecuteMultiple runs multiple SQL statements in sequence
func (e *SQLExecutor) ExecuteMultiple(ctx context.Context, statements []Statement, params map[string]string) ([]Row, error) {
	conn, err := e.pool.Take(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer e.pool.Put(conn)

	// Apply custom functions to connection
	if err := e.applyFunctionsToConn(conn); err != nil {
		return nil, fmt.Errorf("failed to apply functions: %w", err)
	}

	parser := NewSQLParser()
	var lastSelectResult []Row

	for _, stmt := range statements {
		// Bind parameters
		sql := parser.BindParams(stmt.SQL, params)

		result, err := e.executeOnConn(conn, sql)
		if err != nil {
			return nil, fmt.Errorf("execute failed for '%s': %w", truncateSQL(sql), err)
		}

		// Keep track of SELECT results for component rendering
		if stmt.IsSelect && len(result.Rows) > 0 {
			lastSelectResult = result.Rows
		}
	}

	return lastSelectResult, nil
}

// truncateSQL truncates SQL for error messages
func truncateSQL(sql string) string {
	if len(sql) > 50 {
		return sql[:50] + "..."
	}
	return sql
}

// GetConnection gets a raw connection for advanced operations
func (e *SQLExecutor) GetConnection(ctx context.Context) (*sqlite.Conn, func(), error) {
	conn, err := e.pool.Take(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Apply custom functions to connection
	if err := e.applyFunctionsToConn(conn); err != nil {
		e.pool.Put(conn)
		return nil, nil, fmt.Errorf("failed to apply functions: %w", err)
	}

	return conn, func() { e.pool.Put(conn) }, nil
}
