package studio

import (
	"archive/zip"
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestExportSchemaSQL(t *testing.T) {
	schema := &SchemaInfo{
		Driver: "sqlite",
		Tables: []TableInfo{
			{
				Name: "users",
				Columns: []ColumnInfo{
					{Name: "id", Type: "INTEGER", IsPrimaryKey: true},
					{Name: "name", Type: "VARCHAR(100)", IsNullable: false},
				},
			},
		},
	}
	result := ExportSchemaSQL(schema)
	if !strings.Contains(result, "CREATE TABLE") {
		t.Error("expected CREATE TABLE in SQL output")
	}
	if !strings.Contains(result, "users") {
		t.Error("expected table name in SQL output")
	}
	if !strings.Contains(result, "PRIMARY KEY") {
		t.Error("expected PRIMARY KEY in SQL output")
	}
}

func TestExportSchemaJSON(t *testing.T) {
	schema := &SchemaInfo{
		Driver: "sqlite",
		Tables: []TableInfo{
			{Name: "users", Columns: []ColumnInfo{{Name: "id", Type: "INTEGER"}}},
		},
	}
	data, err := ExportSchemaJSON(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(data), "users") {
		t.Error("expected 'users' in JSON output")
	}
}

func TestExportSchemaYAML(t *testing.T) {
	schema := &SchemaInfo{
		Driver: "sqlite",
		Tables: []TableInfo{
			{Name: "users", Columns: []ColumnInfo{{Name: "id", Type: "INTEGER"}}},
		},
	}
	data, err := ExportSchemaYAML(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(data), "users") {
		t.Error("expected 'users' in YAML output")
	}
}

func TestExportSchemaDBML(t *testing.T) {
	schema := &SchemaInfo{
		Tables: []TableInfo{
			{
				Name: "users",
				Columns: []ColumnInfo{
					{Name: "id", Type: "integer", IsPrimaryKey: true},
					{Name: "name", Type: "varchar(100)"},
					{Name: "post_id", Type: "integer", IsForeignKey: true, ForeignTable: "posts", ForeignKey: "id"},
				},
			},
		},
	}
	result := ExportSchemaDBML(schema)
	if !strings.Contains(result, "Table users") {
		t.Error("expected 'Table users' in DBML output")
	}
	if !strings.Contains(result, "[pk") {
		t.Error("expected pk attribute in DBML output")
	}
	if !strings.Contains(result, "Ref:") {
		t.Error("expected Ref: in DBML output")
	}
}

func TestExportSchemaEndpoint(t *testing.T) {
	router, _ := setupTestRouter(t)

	formats := []string{"sql", "json", "yaml", "dbml"}
	for _, format := range formats {
		w := doRequest(router, "GET", "/studio/api/export/schema?format="+format, nil)
		if w.Code != http.StatusOK {
			t.Errorf("export schema format=%s: expected 200, got %d: %s", format, w.Code, w.Body.String())
		}
	}
}

func TestExportGoModelsEndpoint(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/export/models", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "package models") {
		t.Error("expected 'package models' in Go models output")
	}
	if !strings.Contains(body, "struct") {
		t.Error("expected 'struct' in Go models output")
	}
}

func TestExportAllDataJSONEndpoint(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/export/data?format=json", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if result["tables"] == nil {
		t.Error("expected 'tables' key in JSON export")
	}
}

func TestExportAllDataCSVEndpoint(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/export/data?format=csv", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's a valid ZIP
	reader, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatalf("expected valid ZIP file: %v", err)
	}
	if len(reader.File) == 0 {
		t.Error("expected at least one file in ZIP")
	}
}

func TestExportAllDataSQLEndpoint(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/export/data?format=sql", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "INSERT INTO") {
		t.Error("expected INSERT INTO statements in SQL export")
	}
}

func TestExportSchemaERDPNG(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/export/schema?format=png", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Check PNG magic bytes
	body := w.Body.Bytes()
	if len(body) < 4 || body[0] != 0x89 || body[1] != 0x50 || body[2] != 0x4E || body[3] != 0x47 {
		t.Error("expected PNG file (magic bytes \\x89PNG)")
	}
}

func TestExportSchemaERDPDF(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/export/schema?format=pdf", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.HasPrefix(body, "%PDF") {
		t.Error("expected PDF file (magic bytes %%PDF)")
	}
}

func TestRenderERDEmptySchema(t *testing.T) {
	schema := &SchemaInfo{Tables: []TableInfo{}}
	var buf bytes.Buffer
	err := RenderERDPNG(schema, &buf)
	if err != nil {
		t.Fatalf("unexpected error rendering empty schema: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty PNG output for empty schema")
	}
}
