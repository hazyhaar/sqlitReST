package engine

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Statement represents a single SQL statement with optional metadata
type Statement struct {
	SQL      string
	IsSelect bool
}

// SQLParser handles parsing of SQL files
type SQLParser struct{}

// NewSQLParser creates a new SQL parser
func NewSQLParser() *SQLParser {
	return &SQLParser{}
}

// ParseFile reads a SQL file and splits it into individual statements
func (p *SQLParser) ParseFile(filepath string) ([]Statement, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return p.Parse(string(content))
}

// Parse splits SQL content into individual statements
func (p *SQLParser) Parse(content string) ([]Statement, error) {
	var statements []Statement
	var current strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(content))
	inString := false
	stringChar := rune(0)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments at the start
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			if current.Len() > 0 {
				current.WriteString("\n")
			}
			continue
		}

		// Process character by character for proper string handling
		for i, char := range line {
			if !inString {
				if char == '\'' || char == '"' {
					inString = true
					stringChar = char
				} else if char == ';' {
					// End of statement
					current.WriteRune(char)
					stmt := strings.TrimSpace(current.String())
					if stmt != "" && stmt != ";" {
						statements = append(statements, Statement{
							SQL:      stmt,
							IsSelect: p.isSelectStatement(stmt),
						})
					}
					current.Reset()
					// Skip any remaining whitespace on this line after ;
					remaining := strings.TrimSpace(line[i+1:])
					if remaining != "" && !strings.HasPrefix(remaining, "--") {
						current.WriteString(remaining)
					}
					break
				}
			} else {
				if char == stringChar {
					// Check for escaped quote
					if i+1 < len(line) && rune(line[i+1]) == stringChar {
						current.WriteRune(char)
						continue
					}
					inString = false
				}
			}
			current.WriteRune(char)
		}

		if current.Len() > 0 && !strings.HasSuffix(current.String(), ";") {
			current.WriteString("\n")
		}
	}

	// Handle last statement without semicolon
	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		statements = append(statements, Statement{
			SQL:      stmt,
			IsSelect: p.isSelectStatement(stmt),
		})
	}

	return statements, scanner.Err()
}

// isSelectStatement checks if a statement is a SELECT query
func (p *SQLParser) isSelectStatement(sql string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	return strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH")
}

// BindParams replaces :param or $param placeholders with actual values
func (p *SQLParser) BindParams(sql string, params map[string]string) string {
	result := sql

	// Replace :param style
	colonRegex := regexp.MustCompile(`:(\w+)`)
	result = colonRegex.ReplaceAllStringFunc(result, func(match string) string {
		paramName := match[1:] // Remove the :
		if val, ok := params[paramName]; ok {
			return "'" + strings.ReplaceAll(val, "'", "''") + "'"
		}
		return "NULL"
	})

	// Replace $param style
	dollarRegex := regexp.MustCompile(`\$(\w+)`)
	result = dollarRegex.ReplaceAllStringFunc(result, func(match string) string {
		paramName := match[1:] // Remove the $
		if val, ok := params[paramName]; ok {
			return "'" + strings.ReplaceAll(val, "'", "''") + "'"
		}
		return "NULL"
	})

	return result
}
