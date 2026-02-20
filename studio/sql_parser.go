package studio

import (
	"regexp"
	"strings"
)

// ParseCreateStatements parses SQL DDL containing CREATE TABLE statements
// and returns a slice of TableInfo structs.
func ParseCreateStatements(sql string, dialect string) ([]TableInfo, error) {
	if dialect == "" {
		dialect = detectDialect(sql)
	}

	cleaned := removeComments(sql)
	stmts := splitStatements(cleaned)

	var tables []TableInfo
	for _, stmt := range stmts {
		trimmed := strings.TrimSpace(stmt)
		upper := strings.ToUpper(trimmed)
		if !strings.HasPrefix(upper, "CREATE TABLE") {
			continue
		}
		table, err := parseCreateTable(trimmed, dialect)
		if err != nil {
			continue
		}
		tables = append(tables, *table)
	}
	return tables, nil
}

// parseCreateTable parses a single CREATE TABLE statement.
func parseCreateTable(stmt string, dialect string) (*TableInfo, error) {
	// Extract table name
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + "`?" + `"?(\w+)"?` + "`?" + `\s*\(`)
	match := re.FindStringSubmatch(stmt)
	if match == nil {
		return nil, errParseFailed("cannot extract table name")
	}
	tableName := match[1]

	// Extract the body between the outermost parentheses
	body := extractParenBody(stmt)
	if body == "" {
		return nil, errParseFailed("cannot extract column definitions")
	}

	// Split column definitions, respecting nested parentheses
	defs := splitColumnDefs(body)

	table := &TableInfo{
		Name: tableName,
	}

	// Track primary keys declared in constraints
	var constraintPKs []string

	for _, def := range defs {
		def = strings.TrimSpace(def)
		upper := strings.ToUpper(def)

		// Check if this is a table constraint rather than a column definition
		if strings.HasPrefix(upper, "PRIMARY KEY") {
			constraintPKs = append(constraintPKs, extractConstraintColumns(def)...)
			continue
		}
		if strings.HasPrefix(upper, "FOREIGN KEY") {
			localCol, foreignTable, foreignCol := parseForeignKeyConstraint(def)
			if localCol != "" {
				markForeignKey(table, localCol, foreignTable, foreignCol)
			}
			continue
		}
		if strings.HasPrefix(upper, "UNIQUE") || strings.HasPrefix(upper, "CHECK") ||
			strings.HasPrefix(upper, "CONSTRAINT") || strings.HasPrefix(upper, "INDEX") ||
			strings.HasPrefix(upper, "KEY") {
			continue
		}

		col, err := parseColumnDef(def, dialect)
		if err != nil {
			continue
		}
		table.Columns = append(table.Columns, *col)
	}

	// Apply constraint-declared primary keys
	for _, pkName := range constraintPKs {
		for i, col := range table.Columns {
			if strings.EqualFold(col.Name, pkName) {
				table.Columns[i].IsPrimaryKey = true
				table.Columns[i].IsNullable = false
			}
		}
	}

	// Build PrimaryKeys list
	for _, col := range table.Columns {
		if col.IsPrimaryKey {
			table.PrimaryKeys = append(table.PrimaryKeys, col.Name)
		}
	}

	return table, nil
}

// parseColumnDef parses a single column definition.
func parseColumnDef(def string, dialect string) (*ColumnInfo, error) {
	def = strings.TrimSpace(def)
	tokens := tokenizeSQL(def)
	if len(tokens) < 2 {
		return nil, errParseFailed("too few tokens in column def")
	}

	col := &ColumnInfo{
		Name:       unquoteIdent(tokens[0]),
		IsNullable: true,
	}

	// Build the type, handling size specifiers like VARCHAR(255) or DECIMAL(10,2)
	colType := tokens[1]
	idx := 2
	if idx < len(tokens) && tokens[idx] == "(" {
		depth := 1
		colType += "("
		idx++
		for idx < len(tokens) && depth > 0 {
			if tokens[idx] == "(" {
				depth++
			} else if tokens[idx] == ")" {
				depth--
			}
			colType += tokens[idx]
			idx++
		}
	}
	col.Type = colType

	// Parse modifiers
	upper := strings.ToUpper(def)
	if strings.Contains(upper, "PRIMARY KEY") {
		col.IsPrimaryKey = true
		col.IsNullable = false
	}
	if strings.Contains(upper, "NOT NULL") {
		col.IsNullable = false
	}
	if strings.Contains(upper, "AUTOINCREMENT") || strings.Contains(upper, "AUTO_INCREMENT") {
		// Auto-increment is implied by the type for GORM
	}
	if strings.EqualFold(colType, "SERIAL") || strings.EqualFold(colType, "BIGSERIAL") || strings.EqualFold(colType, "SMALLSERIAL") {
		col.IsPrimaryKey = true
		col.IsNullable = false
	}

	// Extract DEFAULT value
	defaultRe := regexp.MustCompile(`(?i)DEFAULT\s+(.+?)(?:\s+(?:NOT\s+NULL|NULL|PRIMARY|UNIQUE|CHECK|REFERENCES|CONSTRAINT)|$)`)
	if dm := defaultRe.FindStringSubmatch(def); dm != nil {
		col.Default = strings.TrimSpace(dm[1])
		// Remove trailing comma or paren
		col.Default = strings.TrimRight(col.Default, ",)")
		col.Default = strings.Trim(col.Default, "'\"")
	}

	// Extract REFERENCES (inline foreign key)
	refRe := regexp.MustCompile(`(?i)REFERENCES\s+` + "`?" + `"?(\w+)"?` + "`?" + `\s*\(\s*` + "`?" + `"?(\w+)"?` + "`?" + `\s*\)`)
	if rm := refRe.FindStringSubmatch(def); rm != nil {
		col.IsForeignKey = true
		col.ForeignTable = rm[1]
		col.ForeignKey = rm[2]
	}

	return col, nil
}

// parseForeignKeyConstraint parses "FOREIGN KEY (col) REFERENCES table(col)".
func parseForeignKeyConstraint(constraint string) (localCol, foreignTable, foreignCol string) {
	re := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(\s*` + "`?" + `"?(\w+)"?` + "`?" + `\s*\)\s*REFERENCES\s+` + "`?" + `"?(\w+)"?` + "`?" + `\s*\(\s*` + "`?" + `"?(\w+)"?` + "`?" + `\s*\)`)
	m := re.FindStringSubmatch(constraint)
	if m == nil {
		return "", "", ""
	}
	return m[1], m[2], m[3]
}

// detectDialect attempts to auto-detect the SQL dialect from content.
func detectDialect(sql string) string {
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "AUTOINCREMENT") {
		return "sqlite"
	}
	if strings.Contains(upper, "SERIAL") || strings.Contains(upper, "::") || strings.Contains(upper, "BYTEA") {
		return "postgres"
	}
	if strings.Contains(upper, "AUTO_INCREMENT") || strings.Contains(upper, "ENGINE=") {
		return "mysql"
	}
	return "sqlite"
}

// --- helpers ---

func removeComments(sql string) string {
	// Remove single-line comments
	lines := strings.Split(sql, "\n")
	var cleaned []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		cleaned = append(cleaned, line)
	}
	result := strings.Join(cleaned, "\n")

	// Remove block comments
	for {
		start := strings.Index(result, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(result[start:], "*/")
		if end < 0 {
			result = result[:start]
			break
		}
		result = result[:start] + result[start+end+2:]
	}
	return result
}

func splitStatements(sql string) []string {
	var stmts []string
	depth := 0
	start := 0
	for i, ch := range sql {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ';':
			if depth == 0 {
				stmt := strings.TrimSpace(sql[start:i])
				if stmt != "" {
					stmts = append(stmts, stmt)
				}
				start = i + 1
			}
		}
	}
	// Trailing statement without semicolon
	if rest := strings.TrimSpace(sql[start:]); rest != "" {
		stmts = append(stmts, rest)
	}
	return stmts
}

func extractParenBody(stmt string) string {
	start := strings.Index(stmt, "(")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(stmt); i++ {
		switch stmt[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return stmt[start+1 : i]
			}
		}
	}
	return ""
}

func splitColumnDefs(body string) []string {
	var defs []string
	depth := 0
	start := 0
	for i, ch := range body {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				def := strings.TrimSpace(body[start:i])
				if def != "" {
					defs = append(defs, def)
				}
				start = i + 1
			}
		}
	}
	if rest := strings.TrimSpace(body[start:]); rest != "" {
		defs = append(defs, rest)
	}
	return defs
}

func tokenizeSQL(s string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar {
				inQuote = false
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}
		switch {
		case ch == '\'' || ch == '"':
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
		case ch == '(' || ch == ')':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(ch))
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		case ch == '`':
			// Skip backticks (MySQL quoting)
			continue
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func unquoteIdent(s string) string {
	s = strings.Trim(s, "`\"'[]")
	return s
}

func extractConstraintColumns(def string) []string {
	re := regexp.MustCompile(`\(\s*(.+?)\s*\)`)
	m := re.FindStringSubmatch(def)
	if m == nil {
		return nil
	}
	var cols []string
	for _, part := range strings.Split(m[1], ",") {
		col := strings.TrimSpace(part)
		col = unquoteIdent(col)
		if col != "" {
			cols = append(cols, col)
		}
	}
	return cols
}

func markForeignKey(table *TableInfo, localCol, foreignTable, foreignCol string) {
	for i, col := range table.Columns {
		if strings.EqualFold(col.Name, localCol) {
			table.Columns[i].IsForeignKey = true
			table.Columns[i].ForeignTable = foreignTable
			table.Columns[i].ForeignKey = foreignCol
		}
	}
}

type parseError struct {
	msg string
}

func (e *parseError) Error() string { return "sql parser: " + e.msg }
func errParseFailed(msg string) error { return &parseError{msg: msg} }
