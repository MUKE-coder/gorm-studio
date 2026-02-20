package studio

import (
	"strings"
	"testing"
)

func TestToGoName(t *testing.T) {
	tests := map[string]string{
		"user_id":    "UserID",
		"created_at": "CreatedAt",
		"name":       "Name",
		"api_key":    "APIKey",
		"http_url":   "HTTPURL",
		"post_tags":  "PostTags",
		"":           "",
	}
	for input, expected := range tests {
		result := toGoName(input)
		if result != expected {
			t.Errorf("toGoName(%q) = %q, want %q", input, result, expected)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := map[string]string{
		"UserID":    "user_i_d",
		"CreatedAt": "created_at",
		"Name":      "name",
	}
	for input, expected := range tests {
		result := toSnakeCase(input)
		if result != expected {
			t.Errorf("toSnakeCase(%q) = %q, want %q", input, result, expected)
		}
	}
}

func TestSqlTypeToGoType(t *testing.T) {
	tests := []struct {
		sqlType  string
		nullable bool
		expected string
	}{
		{"INTEGER", false, "int64"},
		{"INTEGER", true, "*int64"},
		{"VARCHAR(255)", false, "string"},
		{"TEXT", false, "string"},
		{"TEXT", true, "*string"},
		{"BOOLEAN", false, "bool"},
		{"REAL", false, "float64"},
		{"TIMESTAMP", false, "time.Time"},
		{"TIMESTAMP", true, "*time.Time"},
		{"BLOB", false, "[]byte"},
		{"JSONB", false, "json.RawMessage"},
		{"SERIAL", false, "int64"},
		{"UUID", false, "string"},
	}
	for _, tt := range tests {
		result := sqlTypeToGoType(tt.sqlType, tt.nullable)
		if result != tt.expected {
			t.Errorf("sqlTypeToGoType(%q, %v) = %q, want %q", tt.sqlType, tt.nullable, result, tt.expected)
		}
	}
}

func TestBuildGORMTag(t *testing.T) {
	col := ColumnInfo{
		Name:         "id",
		Type:         "INTEGER",
		IsPrimaryKey: true,
		IsNullable:   false,
	}
	tag := buildGORMTag(col)
	if !strings.Contains(tag, "primaryKey") {
		t.Errorf("expected primaryKey in tag, got %q", tag)
	}
	if !strings.Contains(tag, "column:id") {
		t.Errorf("expected column:id in tag, got %q", tag)
	}
}

func TestGenerateGoModels(t *testing.T) {
	schema := &SchemaInfo{
		Tables: []TableInfo{
			{
				Name: "users",
				Columns: []ColumnInfo{
					{Name: "id", Type: "INTEGER", IsPrimaryKey: true},
					{Name: "name", Type: "VARCHAR(100)"},
					{Name: "email", Type: "TEXT", IsNullable: true},
				},
			},
		},
	}
	code := GenerateGoModels(schema)
	if !strings.Contains(code, "package models") {
		t.Error("expected package declaration")
	}
	if !strings.Contains(code, "type User struct") {
		t.Errorf("expected 'type User struct' in output, got:\n%s", code)
	}
	if !strings.Contains(code, "ID") {
		t.Error("expected ID field")
	}
	if !strings.Contains(code, "primaryKey") {
		t.Error("expected primaryKey tag")
	}
}

func TestSingularize(t *testing.T) {
	tests := map[string]string{
		"users":      "user",
		"posts":      "post",
		"categories": "category",
		"addresses":  "address",
		"tag":        "tag",
	}
	for input, expected := range tests {
		result := singularize(input)
		if result != expected {
			t.Errorf("singularize(%q) = %q, want %q", input, result, expected)
		}
	}
}

func TestExtractSize(t *testing.T) {
	tests := map[string]string{
		"VARCHAR(255)":  "255",
		"DECIMAL(10,2)": "10,2",
		"TEXT":          "",
		"INTEGER":       "",
	}
	for input, expected := range tests {
		result := extractSize(input)
		if result != expected {
			t.Errorf("extractSize(%q) = %q, want %q", input, result, expected)
		}
	}
}
