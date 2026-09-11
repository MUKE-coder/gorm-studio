//go:build integration

// Package studio integration tests run the handler surface against real
// PostgreSQL and MySQL servers, catching dialect-specific bugs (quoting, types,
// SQL parsing) that the SQLite unit tests can't. They are gated behind the
// `integration` build tag and read DSNs from the environment:
//
//	STUDIO_PG_DSN    e.g. host=localhost user=postgres password=postgres dbname=studio port=5432 sslmode=disable
//	STUDIO_MYSQL_DSN e.g. root:root@tcp(127.0.0.1:3306)/studio?charset=utf8mb4&parseTime=True&loc=Local
//
// Run: go test ./studio/ -tags integration -run Integration
package studio

import (
	"net/http"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openBackend(t *testing.T, driver, dsn string) *gorm.DB {
	t.Helper()
	var dial gorm.Dialector
	switch driver {
	case "postgres":
		dial = postgres.Open(dsn)
	case "mysql":
		dial = mysql.Open(dsn)
	default:
		t.Fatalf("unknown driver %q", driver)
	}
	db, err := gorm.Open(dial, &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("%s: open: %v", driver, err)
	}
	// Start from a clean slate.
	_ = db.Migrator().DropTable(&TestPost{}, &TestUser{})
	if err := db.AutoMigrate(&TestUser{}, &TestPost{}); err != nil {
		t.Fatalf("%s: migrate: %v", driver, err)
	}
	return db
}

func backends(t *testing.T) map[string]*gorm.DB {
	t.Helper()
	out := map[string]*gorm.DB{}
	if dsn := os.Getenv("STUDIO_PG_DSN"); dsn != "" {
		out["postgres"] = openBackend(t, "postgres", dsn)
	}
	if dsn := os.Getenv("STUDIO_MYSQL_DSN"); dsn != "" {
		out["mysql"] = openBackend(t, "mysql", dsn)
	}
	if len(out) == 0 {
		t.Skip("no integration DSNs set (STUDIO_PG_DSN / STUDIO_MYSQL_DSN)")
	}
	return out
}

func TestIntegration_CRUDAndExport(t *testing.T) {
	for name, db := range backends(t) {
		t.Run(name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			db.Where("1 = 1").Delete(&TestPost{})
			db.Where("1 = 1").Delete(&TestUser{})
			db.Create(&TestUser{Name: "Alice", Email: "alice@int.test", Active: true})
			db.Create(&TestUser{Name: "Bob", Email: "bob@int.test", Active: false})

			router := gin.New()
			if err := Mount(router, db, testModels(), Config{Prefix: "/studio"}); err != nil {
				t.Fatalf("mount: %v", err)
			}

			// List
			w := doRequest(router, "GET", "/studio/api/tables/test_users/rows", nil)
			if w.Code != http.StatusOK {
				t.Fatalf("list: %d %s", w.Code, w.Body.String())
			}
			if total := parseJSON(t, w)["total"].(float64); total != 2 {
				t.Errorf("expected 2 rows, got %v", total)
			}

			// Filter (exercises dialect identifier quoting)
			w = doRequest(router, "GET", "/studio/api/tables/test_users/rows?filter_name=Alice", nil)
			if total := parseJSON(t, w)["total"].(float64); total != 1 {
				t.Errorf("filter: expected 1, got %v", total)
			}

			// Create
			w = doRequest(router, "POST", "/studio/api/tables/test_users/rows",
				map[string]interface{}{"name": "Carol", "email": "carol@int.test"})
			if w.Code != http.StatusCreated {
				t.Fatalf("create: %d %s", w.Code, w.Body.String())
			}

			// SQL read
			w = doRequest(router, "POST", "/studio/api/sql",
				map[string]interface{}{"query": "SELECT COUNT(*) AS n FROM test_users"})
			if w.Code != http.StatusOK {
				t.Fatalf("sql read: %d %s", w.Code, w.Body.String())
			}

			// SQL editor DDL still blocked on this dialect
			w = doRequest(router, "POST", "/studio/api/sql",
				map[string]interface{}{"query": "DROP TABLE test_users"})
			if w.Code != http.StatusForbidden {
				t.Errorf("DDL should be blocked, got %d", w.Code)
			}

			// Export SQL
			w = doRequest(router, "GET", "/studio/api/export/data?format=sql", nil)
			if w.Code != http.StatusOK {
				t.Fatalf("export: %d", w.Code)
			}
		})
	}
}
