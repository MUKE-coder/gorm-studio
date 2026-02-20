package studio

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// ParsedStruct represents a parsed Go struct.
type ParsedStruct struct {
	Name   string
	Fields []ParsedField
}

// ParsedField represents a parsed struct field.
type ParsedField struct {
	Name    string
	GoType  string
	GORMTag string
	JSONTag string
}

// ImportGoModels handles POST /api/import/models
func (h *Handlers) ImportGoModels(c *gin.Context) {
	if h.ReadOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "import not allowed in read-only mode"})
		return
	}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	structs, err := parseGoStructs(string(content))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(structs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no struct definitions found in file"})
		return
	}

	var tablesCreated []string
	var structsParsed []string

	for _, ps := range structs {
		structsParsed = append(structsParsed, ps.Name)
		tableName, err := h.createTableFromStruct(ps)
		if err != nil {
			continue
		}
		tablesCreated = append(tablesCreated, tableName)
	}

	// Refresh schema
	schema, serr := IntrospectSchema(h.DB, h.Models)
	if serr == nil {
		h.Schema = schema
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "models imported successfully",
		"tables_created": tablesCreated,
		"structs_parsed": structsParsed,
	})
}

// parseGoStructs parses Go source code and extracts struct definitions.
func parseGoStructs(source string) ([]ParsedStruct, error) {
	var structs []ParsedStruct

	structRe := regexp.MustCompile(`type\s+(\w+)\s+struct\s*\{`)
	matches := structRe.FindAllStringSubmatchIndex(source, -1)

	for _, match := range matches {
		name := source[match[2]:match[3]]
		braceStart := match[1] - 1

		body := extractBraceBody(source[braceStart:])
		if body == "" {
			continue
		}

		fields := parseStructFields(body)
		if len(fields) > 0 {
			structs = append(structs, ParsedStruct{
				Name:   name,
				Fields: fields,
			})
		}
	}
	return structs, nil
}

// extractBraceBody extracts content between matching braces.
func extractBraceBody(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start+1 : i]
			}
		}
	}
	return ""
}

// parseStructFields parses fields from a struct body.
func parseStructFields(body string) []ParsedField {
	var fields []ParsedField
	lines := strings.Split(body, "\n")

	// Match: FieldName Type `tags`
	fieldRe := regexp.MustCompile("^\\s*(\\w+)\\s+([\\w.*\\[\\]]+)\\s*`(.*)`")
	// Match fields without tags: FieldName Type
	fieldNoTagRe := regexp.MustCompile(`^\s*(\w+)\s+([\w.*\[\]]+)\s*$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		var field ParsedField

		if m := fieldRe.FindStringSubmatch(line); m != nil {
			field.Name = m[1]
			field.GoType = m[2]

			// Extract gorm tag
			gormRe := regexp.MustCompile(`gorm:"([^"]*)"`)
			if gm := gormRe.FindStringSubmatch(m[3]); gm != nil {
				field.GORMTag = gm[1]
			}

			// Extract json tag
			jsonRe := regexp.MustCompile(`json:"([^"]*)"`)
			if jm := jsonRe.FindStringSubmatch(m[3]); jm != nil {
				field.JSONTag = jm[1]
			}
		} else if m := fieldNoTagRe.FindStringSubmatch(line); m != nil {
			field.Name = m[1]
			field.GoType = m[2]
		} else {
			continue
		}

		// Skip relation fields (slices of structs, pointer to structs without basic types)
		if isRelationField(field) {
			continue
		}

		fields = append(fields, field)
	}
	return fields
}

// isRelationField checks if a field represents a GORM relation rather than a column.
func isRelationField(f ParsedField) bool {
	t := f.GoType
	// Slice of non-basic types
	if strings.HasPrefix(t, "[]") {
		inner := t[2:]
		if !isBasicGoType(inner) {
			return true
		}
	}
	// Pointer to non-basic types (but not *string, *int64, etc.)
	if strings.HasPrefix(t, "*") {
		inner := t[1:]
		if !isBasicGoType(inner) && !strings.Contains(inner, ".") {
			return true
		}
	}
	// Non-basic, non-pointer, non-slice types that aren't time.Time etc.
	if !isBasicGoType(t) && !strings.HasPrefix(t, "*") && !strings.HasPrefix(t, "[]") && !strings.Contains(t, ".") {
		return true
	}
	return false
}

// isBasicGoType checks if a type is a basic Go type or common GORM type.
func isBasicGoType(t string) bool {
	t = strings.TrimPrefix(t, "*")
	basics := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true,
		"int64": true, "uint": true, "uint8": true, "uint16": true, "uint32": true,
		"uint64": true, "float32": true, "float64": true, "bool": true,
		"byte": true, "rune": true, "time.Time": true, "json.RawMessage": true,
		"[]byte": true,
	}
	return basics[t]
}

// createTableFromStruct creates a database table from a parsed Go struct.
func (h *Handlers) createTableFromStruct(ps ParsedStruct) (string, error) {
	tableName := toSnakeCase(ps.Name) + "s"
	dialect := h.DB.Dialector.Name()

	var colDefs []string
	for _, f := range ps.Fields {
		colName := toSnakeCase(f.Name)

		// Check gorm tag for column name override
		if f.GORMTag != "" {
			for _, part := range strings.Split(f.GORMTag, ";") {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "column:") {
					colName = strings.TrimPrefix(part, "column:")
				}
			}
		}

		sqlType := goTypeToSQLType(f.GoType, dialect)
		def := quoteIdent(dialect, colName) + " " + sqlType

		if f.GORMTag != "" {
			upperTag := strings.ToUpper(f.GORMTag)
			if strings.Contains(upperTag, "PRIMARYKEY") || strings.Contains(upperTag, "PRIMARY_KEY") {
				def += " PRIMARY KEY"
			}
			if strings.Contains(upperTag, "NOT NULL") {
				def += " NOT NULL"
			}
		}

		colDefs = append(colDefs, def)
	}

	if len(colDefs) == 0 {
		return "", fmt.Errorf("no columns for struct %s", ps.Name)
	}

	ddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n);",
		quoteIdent(dialect, tableName), strings.Join(colDefs, ",\n  "))

	if err := h.DB.Exec(ddl).Error; err != nil {
		return "", fmt.Errorf("creating table %s: %w", tableName, err)
	}
	return tableName, nil
}

// goTypeToSQLType converts a Go type to SQL column type.
func goTypeToSQLType(goType string, dialect string) string {
	t := strings.TrimPrefix(goType, "*")

	switch t {
	case "int", "int64", "uint", "uint64":
		if dialect == "postgres" {
			return "BIGINT"
		}
		return "INTEGER"
	case "int8", "int16", "int32", "uint8", "uint16", "uint32":
		if dialect == "postgres" {
			return "INTEGER"
		}
		return "INTEGER"
	case "float32", "float64":
		if dialect == "postgres" {
			return "DOUBLE PRECISION"
		}
		return "REAL"
	case "bool":
		return "BOOLEAN"
	case "string":
		return "TEXT"
	case "[]byte":
		if dialect == "postgres" {
			return "BYTEA"
		}
		return "BLOB"
	case "time.Time":
		if dialect == "postgres" {
			return "TIMESTAMP"
		}
		return "DATETIME"
	case "json.RawMessage":
		if dialect == "postgres" {
			return "JSONB"
		}
		return "TEXT"
	default:
		return "TEXT"
	}
}
