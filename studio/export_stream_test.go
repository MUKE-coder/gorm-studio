package studio

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupBigRouter seeds more rows than exportBatchSize so exports must page.
func setupBigRouter(t *testing.T, n int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	users := make([]TestUser, n)
	for i := 0; i < n; i++ {
		users[i] = TestUser{Name: "u", Email: fmt.Sprintf("u%d@x", i)}
	}
	if err := db.CreateInBatches(users, 500).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	router := gin.New()
	if err := Mount(router, db, []interface{}{&TestUser{}}, Config{Prefix: "/studio"}); err != nil {
		t.Fatalf("mount: %v", err)
	}
	return router
}

// TestExportStreaming_JSONCountsAllRows verifies the JSON export pages through
// all rows (crossing the exportBatchSize boundary) without loss or duplication.
func TestExportStreaming_JSONCountsAllRows(t *testing.T) {
	const n = 2500 // > 2 * exportBatchSize
	router := setupBigRouter(t, n)

	w := doRequest(router, "GET", "/studio/api/export/data?format=json", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	tables := result["tables"].(map[string]interface{})
	users := tables["test_users"].(map[string]interface{})
	if rc := users["row_count"].(float64); int(rc) != n {
		t.Errorf("expected row_count %d, got %v", n, rc)
	}
	rows := users["rows"].([]interface{})
	if len(rows) != n {
		t.Errorf("expected %d rows in export, got %d", n, len(rows))
	}
}

// TestExportStreaming_SQLCountsAllRows verifies the SQL export emits one INSERT
// per row across batch boundaries.
func TestExportStreaming_SQLCountsAllRows(t *testing.T) {
	const n = 2500
	router := setupBigRouter(t, n)

	w := doRequest(router, "GET", "/studio/api/export/data?format=sql", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := strings.Count(w.Body.String(), "INSERT INTO"); got != n {
		t.Errorf("expected %d INSERT statements, got %d", n, got)
	}
}
