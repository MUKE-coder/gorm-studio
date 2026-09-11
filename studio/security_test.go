package studio

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupReadOnlyRouter mounts studio in read-only mode with a seeded users table.
func setupReadOnlyRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&TestUser{Name: "Alice", Email: "alice@test.com"})
	db.Create(&TestUser{Name: "Bob", Email: "bob@test.com"})
	router := gin.New()
	if err := Mount(router, db, testModels(), Config{Prefix: "/studio", ReadOnly: true}); err != nil {
		t.Fatalf("mount: %v", err)
	}
	return router, db
}

func rowCount(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	db.Table(table).Count(&n)
	return n
}

// --- 0.1 ReadOnly must not be defeated by the SQL-editor read path ---

func TestExecuteSQL_ReadOnlyBlocksStackedWrite(t *testing.T) {
	router, db := setupReadOnlyRouter(t)
	before := rowCount(t, db, "test_users")

	w := doRequest(router, "POST", "/studio/api/sql",
		map[string]interface{}{"query": "SELECT 1; DELETE FROM test_users"})

	if w.Code == http.StatusOK {
		t.Fatalf("stacked statement should be rejected, got 200: %s", w.Body.String())
	}
	if got := rowCount(t, db, "test_users"); got != before {
		t.Errorf("rows changed from %d to %d — stacked DELETE executed", before, got)
	}
}

func TestExecuteSQL_ReadOnlyBlocksPragmaWrite(t *testing.T) {
	router, db := setupReadOnlyRouter(t)

	w := doRequest(router, "POST", "/studio/api/sql",
		map[string]interface{}{"query": "PRAGMA user_version = 42"})

	if w.Code != http.StatusForbidden {
		t.Fatalf("PRAGMA write should be 403 in read-only mode, got %d: %s", w.Code, w.Body.String())
	}
	var v int
	db.Raw("PRAGMA user_version").Scan(&v)
	if v != 0 {
		t.Errorf("user_version changed to %d — PRAGMA write executed", v)
	}
}

func TestExecuteSQL_ReadOnlyAllowsSelect(t *testing.T) {
	router, _ := setupReadOnlyRouter(t)
	w := doRequest(router, "POST", "/studio/api/sql",
		map[string]interface{}{"query": "SELECT * FROM test_users"})
	if w.Code != http.StatusOK {
		t.Fatalf("SELECT should work in read-only mode, got %d: %s", w.Code, w.Body.String())
	}
}

// --- 0.2 DDL blocklist must survive comment / whitespace / stacking / case tricks ---

func TestExecuteSQL_BlocksDDLEvasions(t *testing.T) {
	evasions := []struct {
		name  string
		query string
		// blocked queries return 403; multi-statement inputs return 400.
		// Either way the table must survive.
	}{
		{"block_comment", "/* hi */ DROP TABLE test_users"},
		{"line_comment", "-- c\nDROP TABLE test_users"},
		{"leading_ws", "   \n\t DROP TABLE test_users"},
		{"lowercase", "drop table test_users"},
		{"stacked", "SELECT 1; DROP TABLE test_users"},
		{"mixed_case_comment", "/*x*/DrOp TaBlE test_users"},
	}
	for _, tc := range evasions {
		t.Run(tc.name, func(t *testing.T) {
			router, db := setupTestRouter(t)
			w := doRequest(router, "POST", "/studio/api/sql", map[string]interface{}{"query": tc.query})
			if w.Code == http.StatusOK {
				t.Fatalf("query %q should be rejected, got 200", tc.query)
			}
			var n int64
			db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_users'").Scan(&n)
			if n != 1 {
				t.Errorf("test_users was dropped by %q", tc.query)
			}
		})
	}
}

// --- 0.3 VACUUM INTO must not write an arbitrary file ---

func TestExecuteSQL_BlocksVacuumInto(t *testing.T) {
	router, _ := setupTestRouter(t)
	out := filepath.ToSlash(filepath.Join(t.TempDir(), "stolen.db"))

	w := doRequest(router, "POST", "/studio/api/sql",
		map[string]interface{}{"query": "VACUUM INTO '" + out + "'"})

	if w.Code != http.StatusForbidden {
		t.Fatalf("VACUUM should be 403, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(out); err == nil {
		t.Errorf("VACUUM INTO wrote a file to %s", out)
	}
}

// --- 0.4 SQL data import must run only INSERTs, fail-closed ---

func TestImportDataSQL_RejectsSmuggledDelete(t *testing.T) {
	router, db := setupTestRouter(t)
	before := rowCount(t, db, "test_users")

	// The '(' inside the value used to corrupt paren-based statement splitting,
	// letting the trailing DELETE execute.
	payload := "INSERT INTO test_users (name, email) VALUES ('(', 'z'); DELETE FROM test_users WHERE name = 'Bob';"
	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "x.sql", payload, nil)

	if rec.Code == http.StatusOK {
		t.Fatalf("import with a non-INSERT statement should be rejected, got 200: %s", rec.Body.String())
	}
	if got := rowCount(t, db, "test_users"); got != before {
		t.Errorf("row count changed from %d to %d — smuggled DELETE executed", before, got)
	}
}

func TestImportDataSQL_RejectsPlainDelete(t *testing.T) {
	router, db := setupTestRouter(t)
	before := rowCount(t, db, "test_users")
	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "x.sql",
		"DELETE FROM test_users;", nil)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected rejection, got 200: %s", rec.Body.String())
	}
	if got := rowCount(t, db, "test_users"); got != before {
		t.Errorf("DELETE executed: %d -> %d", before, got)
	}
}

func TestImportDataSQL_AllowsInserts(t *testing.T) {
	router, db := setupTestRouter(t)
	before := rowCount(t, db, "test_users")
	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "x.sql",
		"INSERT INTO test_users (name, email) VALUES ('New', 'new@test.com');", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid INSERT import should succeed, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rowCount(t, db, "test_users"); got != before+1 {
		t.Errorf("expected one inserted row, got %d -> %d", before, got)
	}
}

// --- 0.5 Composite primary keys must be fully specified ---

// setupCompositeRouter creates a table with a two-column primary key.
func setupCompositeRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.Exec(`CREATE TABLE memberships (org_id INTEGER, user_id INTEGER, role TEXT, PRIMARY KEY (org_id, user_id))`)
	db.Exec(`INSERT INTO memberships (org_id, user_id, role) VALUES (1,1,'a'),(1,2,'b'),(2,1,'c')`)
	router := gin.New()
	// No model needed — studio introspects the DB table directly.
	if err := Mount(router, db, []interface{}{}, Config{Prefix: "/studio"}); err != nil {
		t.Fatalf("mount: %v", err)
	}
	return router, db
}

func TestDeleteRow_CompositePKRequiresAllKeys(t *testing.T) {
	router, db := setupCompositeRouter(t)

	// A single value must not delete every row sharing the first key column.
	w := doRequest(router, "DELETE", "/studio/api/tables/memberships/rows/1", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("partial composite key should be 400, got %d: %s", w.Code, w.Body.String())
	}
	if got := rowCount(t, db, "memberships"); got != 3 {
		t.Errorf("rows changed to %d — partial-key delete executed", got)
	}

	// A fully specified composite key deletes exactly one row.
	w = doRequest(router, "DELETE", "/studio/api/tables/memberships/rows/1,1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("full composite key delete should be 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := rowCount(t, db, "memberships"); got != 2 {
		t.Errorf("expected exactly one row deleted, have %d", got)
	}
}

func TestBulkDelete_CompositePKRejected(t *testing.T) {
	router, db := setupCompositeRouter(t)
	w := doRequest(router, "POST", "/studio/api/tables/memberships/rows/bulk-delete",
		map[string]interface{}{"ids": []interface{}{1}})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bulk delete on composite PK should be 400, got %d: %s", w.Code, w.Body.String())
	}
	if got := rowCount(t, db, "memberships"); got != 3 {
		t.Errorf("rows changed to %d — bulk delete executed on composite PK", got)
	}
}
