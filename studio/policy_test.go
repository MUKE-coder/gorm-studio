package studio

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupPolicyRouter mounts studio with the given config over a seeded
// users/posts schema.
func setupPolicyRouter(t *testing.T, cfg Config) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&TestUser{}, &TestPost{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&TestUser{Name: "Alice", Email: "alice@test.com"})
	db.Create(&TestUser{Name: "Bob", Email: "bob@test.com"})
	db.Create(&TestPost{Title: "P1", Body: "b", AuthorID: 1})
	cfg.Prefix = "/studio"
	router := gin.New()
	if err := Mount(router, db, []interface{}{&TestUser{}, &TestPost{}}, cfg); err != nil {
		t.Fatalf("mount: %v", err)
	}
	return router, db
}

// --- 1.1 Scope ---

func TestScope_FiltersRowListing(t *testing.T) {
	// Scope restricts test_users to Alice only.
	router, _ := setupPolicyRouter(t, Config{
		Scope: func(c *gin.Context, table string, tx *gorm.DB) *gorm.DB {
			if table == "test_users" {
				return tx.Where("name = ?", "Alice")
			}
			return tx
		},
	})

	w := doRequest(router, "GET", "/studio/api/tables/test_users/rows", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	result := parseJSON(t, w)
	if total := result["total"].(float64); total != 1 {
		t.Errorf("scope should limit to 1 row, got %v", total)
	}
}

func TestScope_BlocksOutOfScopeDelete(t *testing.T) {
	// Scope only exposes Alice (id 1); deleting Bob (id 2) must affect 0 rows.
	router, db := setupPolicyRouter(t, Config{
		Scope: func(c *gin.Context, table string, tx *gorm.DB) *gorm.DB {
			if table == "test_users" {
				return tx.Where("name = ?", "Alice")
			}
			return tx
		},
	})

	w := doRequest(router, "DELETE", "/studio/api/tables/test_users/rows/2", nil)
	if w.Code == http.StatusOK {
		t.Errorf("delete of out-of-scope row should not succeed, got 200")
	}
	var n int64
	db.Model(&TestUser{}).Where("id = ?", 2).Count(&n)
	if n != 1 {
		t.Errorf("out-of-scope row was deleted despite scope")
	}
}

// --- 1.2 TablePolicy: hidden ---

func TestHiddenTable_OmittedFromSchema(t *testing.T) {
	router, _ := setupPolicyRouter(t, Config{
		TablePolicy: TablePolicy{Hidden: []string{"test_posts"}},
	})
	w := doRequest(router, "GET", "/studio/api/schema", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "test_posts") {
		t.Errorf("hidden table leaked into schema output: %s", w.Body.String())
	}
}

func TestHiddenTable_DirectAccess404(t *testing.T) {
	router, _ := setupPolicyRouter(t, Config{
		TablePolicy: TablePolicy{Hidden: []string{"test_posts"}},
	})
	w := doRequest(router, "GET", "/studio/api/tables/test_posts/rows", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("hidden table should 404 on direct access, got %d", w.Code)
	}
}

func TestHiddenTable_ExcludedFromExport(t *testing.T) {
	router, _ := setupPolicyRouter(t, Config{
		TablePolicy: TablePolicy{Hidden: []string{"test_posts"}},
	})
	w := doRequest(router, "GET", "/studio/api/export/data?format=json", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "test_posts") {
		t.Errorf("hidden table leaked into data export")
	}
	// Schema export too.
	w = doRequest(router, "GET", "/studio/api/export/schema?format=sql", nil)
	if strings.Contains(w.Body.String(), "test_posts") {
		t.Errorf("hidden table leaked into schema export")
	}
}

// --- 1.2 TablePolicy: read-only per table ---

func TestReadOnlyTable_BlocksMutations(t *testing.T) {
	router, db := setupPolicyRouter(t, Config{
		TablePolicy: TablePolicy{ReadOnly: []string{"test_users"}},
	})

	// Reads still work.
	if w := doRequest(router, "GET", "/studio/api/tables/test_users/rows", nil); w.Code != http.StatusOK {
		t.Fatalf("read of read-only table should work, got %d", w.Code)
	}

	// Create / update / delete / bulk-delete are all 403.
	cases := []struct {
		method, path string
		body         interface{}
	}{
		{"POST", "/studio/api/tables/test_users/rows", map[string]interface{}{"name": "Carol", "email": "c@test.com"}},
		{"PUT", "/studio/api/tables/test_users/rows/1", map[string]interface{}{"name": "X"}},
		{"DELETE", "/studio/api/tables/test_users/rows/1", nil},
		{"POST", "/studio/api/tables/test_users/rows/bulk-delete", map[string]interface{}{"ids": []interface{}{1}}},
	}
	for _, tc := range cases {
		w := doRequest(router, tc.method, tc.path, tc.body)
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s on read-only table should be 403, got %d", tc.method, tc.path, w.Code)
		}
	}

	var n int64
	db.Model(&TestUser{}).Count(&n)
	if n != 2 {
		t.Errorf("read-only table was mutated: count=%d", n)
	}
}

func TestReadOnlyTable_BlocksImport(t *testing.T) {
	router, db := setupPolicyRouter(t, Config{
		TablePolicy: TablePolicy{ReadOnly: []string{"test_users"}},
	})
	rec := doMultipartRequest(router, "/studio/api/import/data", "file", "x.csv",
		"name,email\nCarol,c@test.com\n", map[string]string{"table": "test_users"})
	if rec.Code == http.StatusOK {
		t.Errorf("import into read-only table should be rejected, got 200: %s", rec.Body.String())
	}
	var n int64
	db.Model(&TestUser{}).Count(&n)
	if n != 2 {
		t.Errorf("read-only table gained rows via import: count=%d", n)
	}
}

// --- 1.3 Audit log ---

func TestAuditLogger_RecordsMutations(t *testing.T) {
	var events []AuditEvent
	router, _ := setupPolicyRouter(t, Config{
		AuditLogger: func(e AuditEvent) { events = append(events, e) },
	})

	w := doRequest(router, "POST", "/studio/api/tables/test_users/rows",
		map[string]interface{}{"name": "Carol", "email": "c@test.com"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	w = doRequest(router, "DELETE", "/studio/api/tables/test_users/rows/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete failed: %d %s", w.Code, w.Body.String())
	}

	var sawCreate, sawDelete bool
	for _, e := range events {
		if e.Action == "create_row" && e.Table == "test_users" && e.Success {
			sawCreate = true
		}
		if e.Action == "delete_row" && e.Table == "test_users" && e.Success {
			sawDelete = true
		}
	}
	if !sawCreate {
		t.Error("expected a create_row audit event")
	}
	if !sawDelete {
		t.Error("expected a delete_row audit event")
	}
}

func TestAuditLogger_CapturesActor(t *testing.T) {
	var events []AuditEvent
	router, _ := setupPolicyRouter(t, Config{
		AuthMiddleware: func(c *gin.Context) { c.Set("studio_user", "operator@corp"); c.Next() },
		AuditLogger:    func(e AuditEvent) { events = append(events, e) },
	})
	w := doRequest(router, "POST", "/studio/api/tables/test_users/rows",
		map[string]interface{}{"name": "Carol", "email": "c@test.com"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	found := false
	for _, e := range events {
		if e.Action == "create_row" {
			found = true
			if e.Actor != "operator@corp" {
				t.Errorf("expected actor 'operator@corp', got %q", e.Actor)
			}
		}
	}
	if !found {
		t.Error("no create_row event recorded")
	}
}
