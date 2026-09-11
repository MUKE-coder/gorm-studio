package studio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupLimitRouter(t *testing.T, cfg Config) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Pin to a single connection so the in-memory DB (which is per-connection)
	// stays consistent even when the context-cancellation path returns a
	// connection to the pool.
	if sqlDB, derr := db.DB(); derr == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&TestUser{Name: "Alice", Email: "alice@test.com"})
	db.Create(&TestUser{Name: "Bob", Email: "bob@test.com"})
	cfg.Prefix = "/studio"
	router := gin.New()
	if err := Mount(router, db, []interface{}{&TestUser{}}, cfg); err != nil {
		t.Fatalf("mount: %v", err)
	}
	return router, db
}

// --- 2.1 Import size limit ---

func TestImport_MaxBytesRejectsLargeUpload(t *testing.T) {
	router, _ := setupLimitRouter(t, Config{MaxImportBytes: 100})
	big := "name,email\n" + strings.Repeat("x,y\n", 500) // well over 100 bytes
	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "x.csv", big,
		map[string]string{"table": "test_users"})
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized import should be 413, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- 2.1 Import row cap ---

func TestImport_MaxRowsRejectsTooManyRows(t *testing.T) {
	router, db := setupLimitRouter(t, Config{MaxImportRows: 2})
	csv := "name,email\na,a@x\nb,b@x\nc,c@x\nd,d@x\n"
	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "x.csv", csv,
		map[string]string{"table": "test_users"})
	if rec.Code == http.StatusOK {
		t.Errorf("import beyond row cap should be rejected, got 200: %s", rec.Body.String())
	}
	var n int64
	db.Model(&TestUser{}).Count(&n)
	if n > 2+2 { // 2 seeded + at most the cap
		t.Errorf("row cap not enforced: %d rows present", n)
	}
}

// --- 2.1 Import timeout ---

// The import timeout is enforced by binding the import's DB to a context with a
// deadline; actual cancellation depends on the driver honoring the context
// (PostgreSQL/MySQL do; the pure-Go SQLite test driver does not). This test
// verifies the plumbing: a positive ImportTimeout yields a deadline, and a
// negative one does not.
func TestImport_TimeoutSetsDeadline(t *testing.T) {
	_, db := setupLimitRouter(t, Config{})
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/studio/api/import/data", nil)

	withTO := &Handlers{DB: db, ImportTimeout: 5 * time.Second}
	gdb, cancel := withTO.importDB(c)
	defer cancel()
	if _, ok := gdb.Statement.Context.Deadline(); !ok {
		t.Error("expected a deadline on the import DB context when ImportTimeout > 0")
	}

	noTO := &Handlers{DB: db, ImportTimeout: -1}
	gdb2, cancel2 := noTO.importDB(c)
	defer cancel2()
	if _, ok := gdb2.Statement.Context.Deadline(); ok {
		t.Error("expected no deadline when ImportTimeout is disabled")
	}
}

// --- 2.2 Schema import column-type injection ---

func TestImportSchema_RejectsUnsafeColumnType(t *testing.T) {
	router, db := setupLimitRouter(t, Config{})
	payload := `[{"name":"evil","columns":[{"name":"id","type":"INTEGER); DROP TABLE test_users; --","is_primary_key":true}]}]`
	rec := doMultipartRequest(router, "/studio/api/import/schema", "file", "s.json", payload, nil)
	if rec.Code == http.StatusOK {
		t.Errorf("unsafe column type should be rejected, got 200: %s", rec.Body.String())
	}
	var n int64
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_users'").Scan(&n)
	if n != 1 {
		t.Errorf("test_users was dropped via column-type injection")
	}
}

func TestImportSchema_AllowsNormalTypes(t *testing.T) {
	router, db := setupLimitRouter(t, Config{})
	payload := `[{"name":"widgets","columns":[{"name":"id","type":"INTEGER","is_primary_key":true},{"name":"price","type":"DECIMAL(10,2)"},{"name":"label","type":"VARCHAR(255)"}]}]`
	rec := doMultipartRequest(router, "/studio/api/import/schema", "file", "s.json", payload, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid schema import should succeed, got %d: %s", rec.Code, rec.Body.String())
	}
	var n int64
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='widgets'").Scan(&n)
	if n != 1 {
		t.Errorf("widgets table was not created")
	}
}

// --- 2.3 Dry run ---

func TestImportSchema_DryRunCreatesNothing(t *testing.T) {
	router, db := setupLimitRouter(t, Config{})
	payload := `[{"name":"widgets","columns":[{"name":"id","type":"INTEGER","is_primary_key":true}]}]`
	rec := doMultipartRequest(router, "/studio/api/import/schema?dry_run=true", "file", "s.json", payload, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("dry run should be 200, got %d: %s", rec.Code, rec.Body.String())
	}
	result := parseJSON(t, rec)
	if result["dry_run"] != true {
		t.Errorf("expected dry_run:true, got %v", result["dry_run"])
	}
	var n int64
	db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='widgets'").Scan(&n)
	if n != 0 {
		t.Errorf("dry run created the table")
	}
}

func TestImportData_DryRunInsertsNothing(t *testing.T) {
	router, db := setupLimitRouter(t, Config{})
	var before int64
	db.Model(&TestUser{}).Count(&before)
	csv := "name,email\nCarol,c@x\nDave,d@x\n"
	rec := doMultipartRequest(router, "/studio/api/import/data?dry_run=true", "file", "x.csv", csv,
		map[string]string{"table": "test_users"})
	if rec.Code != http.StatusOK {
		t.Fatalf("dry run should be 200, got %d: %s", rec.Code, rec.Body.String())
	}
	result := parseJSON(t, rec)
	if result["dry_run"] != true {
		t.Errorf("expected dry_run:true, got %v", result["dry_run"])
	}
	if rows := result["rows_to_insert"].(float64); rows != 2 {
		t.Errorf("expected rows_to_insert=2, got %v", rows)
	}
	var after int64
	db.Model(&TestUser{}).Count(&after)
	if after != before {
		t.Errorf("dry run inserted rows: %d -> %d", before, after)
	}
}
