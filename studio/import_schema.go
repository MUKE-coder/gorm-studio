package studio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ImportSchema handles POST /api/import/schema
func (h *Handlers) ImportSchema(c *gin.Context) {
	if h.ReadOnly {
		c.JSON(http.StatusForbidden, gin.H{"error": "import not allowed in read-only mode"})
		return
	}

	file, header, err := c.Request.FormFile("file")
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

	ext := strings.ToLower(filepath.Ext(header.Filename))

	var tables []TableInfo
	switch ext {
	case ".sql":
		tables, err = h.importSchemaFromSQL(string(content))
	case ".json":
		tables, err = h.importSchemaFromJSON(content)
	case ".yaml", ".yml":
		tables, err = h.importSchemaFromYAML(content)
	case ".dbml":
		tables, err = h.importSchemaFromDBML(string(content))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format: " + ext + ". Use .sql, .json, .yaml, or .dbml"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create tables in the database
	tablesCreated, err := h.createTablesFromInfo(tables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Refresh schema
	schema, err := IntrospectSchema(h.DB, h.Models)
	if err == nil {
		h.Schema = schema
	}

	// Generate Go code
	goCode := GenerateGoModels(h.Schema)

	c.JSON(http.StatusOK, gin.H{
		"message":        "schema imported successfully",
		"tables_created": tablesCreated,
		"go_code":        goCode,
	})
}

func (h *Handlers) importSchemaFromSQL(content string) ([]TableInfo, error) {
	dialect := h.DB.Dialector.Name()
	tables, err := ParseCreateStatements(content, dialect)
	if err != nil {
		return nil, fmt.Errorf("parsing SQL: %w", err)
	}
	if len(tables) == 0 {
		return nil, fmt.Errorf("no CREATE TABLE statements found")
	}
	return tables, nil
}

func (h *Handlers) importSchemaFromJSON(data []byte) ([]TableInfo, error) {
	// Try parsing as SchemaInfo first
	var schema SchemaInfo
	if err := json.Unmarshal(data, &schema); err == nil && len(schema.Tables) > 0 {
		return schema.Tables, nil
	}

	// Try parsing as array of TableInfo
	var tables []TableInfo
	if err := json.Unmarshal(data, &tables); err == nil && len(tables) > 0 {
		return tables, nil
	}

	return nil, fmt.Errorf("invalid JSON schema format")
}

func (h *Handlers) importSchemaFromYAML(data []byte) ([]TableInfo, error) {
	// Try parsing as SchemaInfo first
	var schema SchemaInfo
	if err := yaml.Unmarshal(data, &schema); err == nil && len(schema.Tables) > 0 {
		return schema.Tables, nil
	}

	// Try parsing as array of TableInfo
	var tables []TableInfo
	if err := yaml.Unmarshal(data, &tables); err == nil && len(tables) > 0 {
		return tables, nil
	}

	return nil, fmt.Errorf("invalid YAML schema format")
}

func (h *Handlers) importSchemaFromDBML(content string) ([]TableInfo, error) {
	return parseDBML(content)
}

// createTablesFromInfo creates database tables from parsed TableInfo slices.
func (h *Handlers) createTablesFromInfo(tables []TableInfo) ([]string, error) {
	dialect := h.DB.Dialector.Name()
	var created []string

	for _, table := range tables {
		ddl := generateCreateTableSQL(table, dialect)
		// Use IF NOT EXISTS to avoid errors on existing tables
		ddl = strings.Replace(ddl, "CREATE TABLE", "CREATE TABLE IF NOT EXISTS", 1)
		if err := h.DB.Exec(ddl).Error; err != nil {
			return created, fmt.Errorf("creating table %s: %w", table.Name, err)
		}
		created = append(created, table.Name)
	}
	return created, nil
}

// parseDBML is a simple DBML parser that extracts table definitions.
func parseDBML(content string) ([]TableInfo, error) {
	var tables []TableInfo
	lines := strings.Split(content, "\n")

	var currentTable *TableInfo
	inTable := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Table definition start
		if strings.HasPrefix(strings.ToLower(line), "table ") && strings.HasSuffix(line, "{") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				tableName := strings.Trim(parts[1], "\"{}`")
				currentTable = &TableInfo{Name: tableName}
				inTable = true
				continue
			}
		}

		// Table definition end
		if line == "}" && inTable {
			if currentTable != nil {
				// Build PrimaryKeys
				for _, col := range currentTable.Columns {
					if col.IsPrimaryKey {
						currentTable.PrimaryKeys = append(currentTable.PrimaryKeys, col.Name)
					}
				}
				tables = append(tables, *currentTable)
			}
			currentTable = nil
			inTable = false
			continue
		}

		// Column definition inside table
		if inTable && currentTable != nil && !strings.HasPrefix(strings.ToLower(line), "ref:") {
			col := parseDBMLColumn(line)
			if col != nil {
				currentTable.Columns = append(currentTable.Columns, *col)
			}
		}

		// Ref definitions (outside tables)
		if strings.HasPrefix(strings.ToLower(line), "ref:") {
			parseDBMLRef(line, &tables)
		}
	}

	if len(tables) == 0 {
		return nil, fmt.Errorf("no tables found in DBML")
	}
	return tables, nil
}

// parseDBMLColumn parses a DBML column line like "id integer [pk, increment]"
func parseDBMLColumn(line string) *ColumnInfo {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// Extract attributes in brackets
	attrs := ""
	if idx := strings.Index(line, "["); idx >= 0 {
		endIdx := strings.Index(line, "]")
		if endIdx > idx {
			attrs = strings.ToLower(line[idx+1 : endIdx])
		}
		line = strings.TrimSpace(line[:idx])
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	col := &ColumnInfo{
		Name:       parts[0],
		Type:       strings.Join(parts[1:], " "),
		IsNullable: true,
	}

	if strings.Contains(attrs, "pk") {
		col.IsPrimaryKey = true
		col.IsNullable = false
	}
	if strings.Contains(attrs, "not null") {
		col.IsNullable = false
	}

	// Extract default value
	if idx := strings.Index(attrs, "default:"); idx >= 0 {
		rest := attrs[idx+8:]
		rest = strings.TrimSpace(rest)
		rest = strings.TrimRight(rest, ",]")
		rest = strings.Trim(rest, "' \"")
		col.Default = rest
	}

	// Extract ref
	if idx := strings.Index(attrs, "ref:"); idx >= 0 {
		rest := attrs[idx+4:]
		rest = strings.TrimSpace(rest)
		rest = strings.TrimLeft(rest, "> <-")
		rest = strings.TrimSpace(rest)
		refParts := strings.Split(rest, ".")
		if len(refParts) == 2 {
			col.IsForeignKey = true
			col.ForeignTable = strings.TrimRight(refParts[0], ",]")
			col.ForeignKey = strings.TrimRight(refParts[1], ",]")
		}
	}

	return col
}

// parseDBMLRef parses a standalone ref line like "Ref: posts.author_id > users.id"
func parseDBMLRef(line string, tables *[]TableInfo) {
	// Format: Ref: table1.col1 > table2.col2
	line = strings.TrimPrefix(line, "Ref:")
	line = strings.TrimPrefix(line, "ref:")
	line = strings.TrimSpace(line)

	// Remove relationship operator
	for _, op := range []string{" > ", " < ", " - "} {
		if strings.Contains(line, op) {
			parts := strings.SplitN(line, op, 2)
			if len(parts) == 2 {
				from := strings.TrimSpace(parts[0])
				to := strings.TrimSpace(parts[1])
				fromParts := strings.Split(from, ".")
				toParts := strings.Split(to, ".")
				if len(fromParts) == 2 && len(toParts) == 2 {
					for i, table := range *tables {
						if table.Name == fromParts[0] {
							for j, col := range table.Columns {
								if col.Name == fromParts[1] {
									(*tables)[i].Columns[j].IsForeignKey = true
									(*tables)[i].Columns[j].ForeignTable = toParts[0]
									(*tables)[i].Columns[j].ForeignKey = toParts[1]
								}
							}
						}
					}
				}
			}
			break
		}
	}
}
