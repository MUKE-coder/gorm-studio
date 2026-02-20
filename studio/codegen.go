package studio

import (
	"fmt"
	"strings"
	"unicode"
)

// GenerateGoModels generates a complete Go source file with struct definitions
// for all tables in the schema.
func GenerateGoModels(schema *SchemaInfo) string {
	var sb strings.Builder
	sb.WriteString("package models\n\n")

	// Collect imports needed
	needsTime := false
	needsJSON := false
	for _, table := range schema.Tables {
		for _, col := range table.Columns {
			goType := sqlTypeToGoType(col.Type, col.IsNullable)
			if col.GoType != "" {
				goType = col.GoType
			}
			if strings.Contains(goType, "time.Time") {
				needsTime = true
			}
			if strings.Contains(goType, "json.RawMessage") {
				needsJSON = true
			}
		}
	}

	if needsTime || needsJSON {
		sb.WriteString("import (\n")
		if needsJSON {
			sb.WriteString("\t\"encoding/json\"\n")
		}
		if needsTime {
			sb.WriteString("\t\"time\"\n")
		}
		sb.WriteString(")\n\n")
	}

	for i, table := range schema.Tables {
		sb.WriteString(generateStructForTable(&table, schema))
		if i < len(schema.Tables)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// generateStructForTable generates a single Go struct definition for a table.
func generateStructForTable(table *TableInfo, schema *SchemaInfo) string {
	var sb strings.Builder

	structName := toGoName(singularize(table.Name))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", structName))

	for _, col := range table.Columns {
		goType := sqlTypeToGoType(col.Type, col.IsNullable)
		if col.GoType != "" {
			goType = col.GoType
		}
		fieldName := toGoName(col.Name)

		gormTag := buildGORMTag(col)
		jsonTag := fmt.Sprintf(`json:"%s"`, col.Name)

		sb.WriteString(fmt.Sprintf("\t%s %s `%s %s`\n", fieldName, goType, gormTag, jsonTag))
	}

	// Add relation fields
	for _, rel := range table.Relations {
		relStructName := toGoName(singularize(rel.Table))
		fieldName := toGoName(rel.Name)

		var fieldType string
		var tag string
		switch rel.Type {
		case "has_one", "belongs_to":
			fieldType = relStructName
			tag = fmt.Sprintf(`gorm:"foreignKey:%s"`, toGoName(rel.ForeignKey))
		case "has_many":
			fieldType = "[]" + relStructName
			tag = fmt.Sprintf(`gorm:"foreignKey:%s"`, toGoName(rel.ForeignKey))
		case "many_to_many":
			fieldType = "[]" + relStructName
			if rel.JoinTable != "" {
				tag = fmt.Sprintf(`gorm:"many2many:%s"`, rel.JoinTable)
			} else {
				tag = fmt.Sprintf(`gorm:"many2many:%s_%s"`, table.Name, rel.Table)
			}
		default:
			continue
		}
		jsonName := toSnakeCase(rel.Name)
		sb.WriteString(fmt.Sprintf("\t%s %s `%s json:\"%s,omitempty\"`\n", fieldName, fieldType, tag, jsonName))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// sqlTypeToGoType maps SQL column types to Go types.
func sqlTypeToGoType(sqlType string, nullable bool) string {
	upper := strings.ToUpper(sqlType)
	// Strip size/precision: VARCHAR(255) -> VARCHAR
	base := upper
	if idx := strings.Index(base, "("); idx >= 0 {
		base = base[:idx]
	}
	base = strings.TrimSpace(base)

	var goType string
	switch base {
	case "INTEGER", "INT", "BIGINT", "INT8", "INT4":
		goType = "int64"
	case "SMALLINT", "TINYINT", "INT2":
		goType = "int32"
	case "REAL", "FLOAT", "DOUBLE", "DOUBLE PRECISION", "FLOAT8", "FLOAT4":
		goType = "float64"
	case "NUMERIC", "DECIMAL":
		goType = "float64"
	case "BOOLEAN", "BOOL":
		goType = "bool"
	case "TEXT", "VARCHAR", "CHAR", "CHARACTER", "CHARACTER VARYING", "NVARCHAR", "LONGTEXT", "MEDIUMTEXT", "TINYTEXT", "STRING":
		goType = "string"
	case "BLOB", "BYTEA", "BINARY", "VARBINARY", "LONGBLOB", "MEDIUMBLOB":
		return "[]byte"
	case "TIMESTAMP", "DATETIME", "DATE", "TIMESTAMPTZ", "TIMESTAMP WITHOUT TIME ZONE", "TIMESTAMP WITH TIME ZONE":
		goType = "time.Time"
	case "UUID":
		goType = "string"
	case "JSON", "JSONB":
		return "json.RawMessage"
	case "SERIAL", "BIGSERIAL", "SMALLSERIAL":
		goType = "int64"
	default:
		goType = "string"
	}

	if nullable && goType != "[]byte" && goType != "json.RawMessage" {
		return "*" + goType
	}
	return goType
}

// toGoName converts a snake_case or lowercase name to PascalCase Go name.
func toGoName(name string) string {
	if name == "" {
		return ""
	}

	// Common acronyms that should be all-caps
	acronyms := map[string]string{
		"id": "ID", "url": "URL", "api": "API", "ip": "IP",
		"http": "HTTP", "https": "HTTPS", "sql": "SQL", "ssh": "SSH",
		"uuid": "UUID", "uri": "URI", "html": "HTML", "css": "CSS",
		"json": "JSON", "xml": "XML", "cpu": "CPU", "gpu": "GPU",
	}

	parts := strings.Split(name, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		if acronym, ok := acronyms[lower]; ok {
			result.WriteString(acronym)
		} else {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}
	return result.String()
}

// toSnakeCase converts PascalCase or camelCase to snake_case.
func toSnakeCase(name string) string {
	var result strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// buildGORMTag constructs the gorm:"..." tag string from a ColumnInfo.
func buildGORMTag(col ColumnInfo) string {
	var parts []string

	if col.IsPrimaryKey {
		parts = append(parts, "primaryKey")
	}
	parts = append(parts, "column:"+col.Name)

	if size := extractSize(col.Type); size != "" {
		parts = append(parts, "size:"+size)
	}

	if !col.IsNullable && !col.IsPrimaryKey {
		parts = append(parts, "not null")
	}

	if col.Default != "" {
		parts = append(parts, "default:"+col.Default)
	}

	return `gorm:"` + strings.Join(parts, ";") + `"`
}

// extractSize extracts the size from a SQL type like VARCHAR(255) -> "255".
func extractSize(sqlType string) string {
	start := strings.Index(sqlType, "(")
	end := strings.Index(sqlType, ")")
	if start >= 0 && end > start {
		return sqlType[start+1 : end]
	}
	return ""
}

// singularize does a simple singularization of a table name.
func singularize(name string) string {
	if strings.HasSuffix(name, "ies") {
		return name[:len(name)-3] + "y"
	}
	if strings.HasSuffix(name, "ses") || strings.HasSuffix(name, "xes") || strings.HasSuffix(name, "zes") {
		return name[:len(name)-2]
	}
	if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
		return name[:len(name)-1]
	}
	return name
}
