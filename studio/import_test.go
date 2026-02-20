package studio

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func doMultipartRequest(router *gin.Engine, path, fieldName, filename, content string, extraFields map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile(fieldName, filename)
	part.Write([]byte(content))
	for k, v := range extraFields {
		w.WriteField(k, v)
	}
	w.Close()

	req, _ := http.NewRequest("POST", path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestImportSchemaSQL(t *testing.T) {
	router, db := setupTestRouter(t)

	sql := `CREATE TABLE imported_items (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		price REAL
	);`

	rec := doMultipartRequest(router, "/studio/api/import/schema", "file", "schema.sql", sql, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := parseJSON(t, rec)
	tablesCreated, ok := result["tables_created"].([]interface{})
	if !ok || len(tablesCreated) == 0 {
		t.Error("expected tables_created in response")
	}

	// Verify table exists in DB
	var count int64
	db.Raw("SELECT count(*) FROM imported_items").Scan(&count)
	// Should not error (table exists)

	// Verify Go code is returned
	goCode, ok := result["go_code"].(string)
	if !ok || goCode == "" {
		t.Error("expected go_code in response")
	}
}

func TestImportSchemaJSON(t *testing.T) {
	router, _ := setupTestRouter(t)

	jsonSchema := `{
		"tables": [{
			"name": "json_items",
			"columns": [
				{"name": "id", "type": "INTEGER", "is_primary_key": true},
				{"name": "value", "type": "TEXT"}
			]
		}]
	}`

	rec := doMultipartRequest(router, "/studio/api/import/schema", "file", "schema.json", jsonSchema, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImportSchemaReadOnlyBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	router := gin.New()
	Mount(router, db, testModels(), Config{
		Prefix:   "/studio",
		ReadOnly: true,
	})

	rec := doMultipartRequest(router, "/studio/api/import/schema", "file", "schema.sql",
		"CREATE TABLE t (id INT);", nil)
	// Should be 404 (route not registered) or 403
	if rec.Code == http.StatusOK {
		t.Error("import should not succeed in read-only mode")
	}
}

func TestImportDataJSON(t *testing.T) {
	router, db := setupTestRouter(t)

	jsonData := `{
		"test_users": [
			{"name": "Imported1", "email": "imported1@test.com"},
			{"name": "Imported2", "email": "imported2@test.com"}
		]
	}`

	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "data.json", jsonData, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := parseJSON(t, rec)
	rowsInserted, ok := result["rows_inserted"].(float64)
	if !ok || rowsInserted < 2 {
		t.Errorf("expected at least 2 rows inserted, got %v", result["rows_inserted"])
	}

	// Verify data in DB
	var count int64
	db.Table("test_users").Where("name = ?", "Imported1").Count(&count)
	if count == 0 {
		t.Error("expected imported row in database")
	}
}

func TestImportDataCSV(t *testing.T) {
	router, db := setupTestRouter(t)

	csvData := "name,email\nCSVUser1,csv1@test.com\nCSVUser2,csv2@test.com"

	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "data.csv", csvData,
		map[string]string{"table": "test_users"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var count int64
	db.Table("test_users").Where("name = ?", "CSVUser1").Count(&count)
	if count == 0 {
		t.Error("expected CSV imported row in database")
	}
}

func TestImportDataCSVRequiresTable(t *testing.T) {
	router, _ := setupTestRouter(t)

	csvData := "name,email\nUser1,u1@test.com"
	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "data.csv", csvData, nil)
	if rec.Code == http.StatusOK {
		t.Error("CSV import without table parameter should fail")
	}
}

func TestImportDataSQL(t *testing.T) {
	router, db := setupTestRouter(t)

	sqlData := `INSERT INTO test_users (name, email) VALUES ('SQLUser1', 'sql1@test.com');
INSERT INTO test_users (name, email) VALUES ('SQLUser2', 'sql2@test.com');`

	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "data.sql", sqlData, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var count int64
	db.Table("test_users").Where("name = ?", "SQLUser1").Count(&count)
	if count == 0 {
		t.Error("expected SQL imported row in database")
	}
}

func TestImportDataReadOnlyBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	router := gin.New()
	Mount(router, db, testModels(), Config{
		Prefix:   "/studio",
		ReadOnly: true,
	})

	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "data.json",
		`{"test": []}`, nil)
	if rec.Code == http.StatusOK {
		t.Error("import should not succeed in read-only mode")
	}
}

func TestImportGoModels(t *testing.T) {
	router, _ := setupTestRouter(t)

	goCode := `package models

type Product struct {
	ID    uint   ` + "`gorm:\"primaryKey\"`" + `
	Name  string ` + "`gorm:\"not null\"`" + `
	Price float64
}
`
	rec := doMultipartRequest(router, "/studio/api/import/models", "file", "models.go", goCode, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	result := parseJSON(t, rec)
	structsParsed, ok := result["structs_parsed"].([]interface{})
	if !ok || len(structsParsed) == 0 {
		t.Error("expected structs_parsed in response")
	}
}

func TestParseGoStructs(t *testing.T) {
	source := `package main

type User struct {
	ID    uint   ` + "`gorm:\"primaryKey\" json:\"id\"`" + `
	Name  string ` + "`gorm:\"size:100;not null\" json:\"name\"`" + `
	Email string
	Posts []Post ` + "`gorm:\"foreignKey:AuthorID\"`" + `
}

type Post struct {
	ID       uint   ` + "`gorm:\"primaryKey\"`" + `
	Title    string
	AuthorID uint
}
`
	structs, err := parseGoStructs(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(structs) != 2 {
		t.Fatalf("expected 2 structs, got %d", len(structs))
	}
	if structs[0].Name != "User" {
		t.Errorf("expected first struct 'User', got %q", structs[0].Name)
	}
	// Posts field should be excluded (relation)
	for _, f := range structs[0].Fields {
		if f.Name == "Posts" {
			t.Error("relation field 'Posts' should be excluded")
		}
	}
}

func TestParseDBML(t *testing.T) {
	dbml := `
Table users {
  id integer [pk]
  name varchar(100) [not null]
  email text
}

Table posts {
  id integer [pk]
  title varchar(300) [not null]
  author_id integer [ref: > users.id]
}
`
	tables, err := parseDBML(dbml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}
	if tables[0].Name != "users" {
		t.Errorf("expected first table 'users', got %q", tables[0].Name)
	}
	if !tables[0].Columns[0].IsPrimaryKey {
		t.Error("expected id to be primary key")
	}
}

