package parser

import (
	"strings"
	"unicode"
)

// MySQLValue represents a parsed value from a MySQL INSERT statement.
// nil means SQL NULL.
type MySQLValue = *string

// NullValue returns a nil MySQLValue (SQL NULL).
func NullValue() MySQLValue {
	return nil
}

// StringValue returns a MySQLValue wrapping the given string.
func StringValue(s string) MySQLValue {
	return &s
}

// ExtractStatements splits a MySQL dump into individual SQL statements.
// It is quote-aware: semicolons inside string literals are not treated as delimiters.
func ExtractStatements(sql string) []string {
	var statements []string
	i := 0
	length := len(sql)

	for i < length {
		// Skip whitespace
		for i < length && unicode.IsSpace(rune(sql[i])) {
			i++
		}
		if i >= length {
			break
		}

		// Skip single-line comments (-- ...)
		if i+1 < length && sql[i] == '-' && sql[i+1] == '-' {
			for i < length && sql[i] != '\n' {
				i++
			}
			continue
		}

		// Skip block comments (/* ... */)
		if i+1 < length && sql[i] == '/' && sql[i+1] == '*' {
			end := strings.Index(sql[i+2:], "*/")
			if end == -1 {
				i = length
			} else {
				i = i + 2 + end + 2
			}
			continue
		}

		// Read a statement until unquoted semicolon
		stmtStart := i
		inQuote := false
		for i < length {
			ch := sql[i]
			if ch == '\'' && !inQuote {
				inQuote = true
				i++
			} else if ch == '\'' && inQuote {
				// Check for escaped quote ('')
				if i+1 < length && sql[i+1] == '\'' {
					i += 2
				} else if i > 0 && sql[i-1] == '\\' {
					// Backslash-escaped quote
					i++
				} else {
					inQuote = false
					i++
				}
			} else if ch == ';' && !inQuote {
				statements = append(statements, sql[stmtStart:i+1])
				i++
				break
			} else {
				i++
			}
		}
		// Handle unterminated statement at end of input
		if i >= length {
			remaining := strings.TrimSpace(sql[stmtStart:])
			if remaining != "" {
				statements = append(statements, remaining)
			}
		}
	}

	return statements
}

// ParseMySQLValues parses a MySQL INSERT INTO ... VALUES (...), (...); statement
// and returns the rows as slices of MySQLValue (nil = NULL).
func ParseMySQLValues(stmt string) [][]MySQLValue {
	// Find the VALUES keyword
	upper := strings.ToUpper(stmt)
	idx := strings.Index(upper, "VALUES")
	if idx == -1 {
		return nil
	}
	valuesStr := stmt[idx+6:]

	var rows [][]MySQLValue
	i := 0
	length := len(valuesStr)

	for i < length {
		// Skip to opening paren
		for i < length && valuesStr[i] != '(' {
			i++
		}
		if i >= length {
			break
		}
		i++ // skip '('

		// Parse one row
		var fields []MySQLValue
		for i < length && valuesStr[i] != ')' {
			ch := valuesStr[i]
			if ch == '\'' {
				// Quoted string value
				i++ // skip opening quote
				var val strings.Builder
				for i < length {
					if valuesStr[i] == '\\' && i+1 < length {
						val.WriteByte(valuesStr[i+1])
						i += 2
					} else if valuesStr[i] == '\'' && i+1 < length && valuesStr[i+1] == '\'' {
						val.WriteByte('\'')
						i += 2
					} else if valuesStr[i] == '\'' {
						i++ // skip closing quote
						break
					} else {
						val.WriteByte(valuesStr[i])
						i++
					}
				}
				s := val.String()
				fields = append(fields, &s)
			} else if ch == ' ' || ch == ',' || ch == '\t' || ch == '\n' || ch == '\r' {
				i++
			} else if i+3 < length && strings.EqualFold(valuesStr[i:i+4], "NULL") {
				fields = append(fields, nil)
				i += 4
			} else {
				// Unquoted value (number, etc.)
				var val strings.Builder
				for i < length && valuesStr[i] != ',' && valuesStr[i] != ')' {
					val.WriteByte(valuesStr[i])
					i++
				}
				s := strings.TrimSpace(val.String())
				fields = append(fields, &s)
			}
		}
		rows = append(rows, fields)
		if i < length {
			i++ // skip ')'
		}
	}

	return rows
}

// GetInsertTable extracts the table name from an INSERT INTO statement.
// Returns empty string if not found.
func GetInsertTable(stmt string) string {
	upper := strings.ToUpper(stmt)
	idx := strings.Index(upper, "INSERT INTO")
	if idx == -1 {
		return ""
	}

	rest := strings.TrimSpace(stmt[idx+11:])
	// Table name may be backtick-quoted or plain
	if len(rest) > 0 && rest[0] == '`' {
		end := strings.Index(rest[1:], "`")
		if end == -1 {
			return ""
		}
		return rest[1 : end+1]
	}

	// Plain table name (up to space or paren)
	var name strings.Builder
	for _, ch := range rest {
		if ch == ' ' || ch == '(' || ch == '\t' {
			break
		}
		name.WriteRune(ch)
	}
	return name.String()
}

// ValStr safely dereferences a MySQLValue, returning empty string for NULL.
func ValStr(v MySQLValue) string {
	if v == nil {
		return ""
	}
	return *v
}
