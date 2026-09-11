package studio

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupHeaderRouter(t *testing.T, cfg Config) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&TestUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&TestUser{Name: "Alice", Email: "a@x"})
	cfg.Prefix = "/studio"
	router := gin.New()
	if err := Mount(router, db, []interface{}{&TestUser{}}, cfg); err != nil {
		t.Fatalf("mount: %v", err)
	}
	return router, db
}

// --- 3.1 Security headers & CSP ---

func TestSecurityHeaders_OnHTMLPage(t *testing.T) {
	router, _ := setupHeaderRouter(t, Config{})
	w := doRequest(router, "GET", "/studio", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected a Content-Security-Policy on the HTML page")
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP should forbid framing, got %q", csp)
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options: nosniff")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("expected X-Frame-Options: DENY")
	}
}

func TestSecurityHeaders_OnAPIResponses(t *testing.T) {
	router, _ := setupHeaderRouter(t, Config{})
	w := doRequest(router, "GET", "/studio/api/schema", nil)
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected nosniff on API responses")
	}
}

// --- 3.3 Rate limiting ---

func TestRateLimit_SQLEndpoint(t *testing.T) {
	router, _ := setupHeaderRouter(t, Config{
		RateLimit: RateLimitConfig{SQLPerMinute: 3},
	})

	var got429 bool
	for i := 0; i < 6; i++ {
		w := doRequest(router, "POST", "/studio/api/sql", map[string]interface{}{"query": "SELECT 1"})
		if w.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Error("expected a 429 after exceeding the SQL rate limit")
	}
}

func TestRateLimit_Unlimited_WhenUnset(t *testing.T) {
	router, _ := setupHeaderRouter(t, Config{}) // no RateLimit
	for i := 0; i < 20; i++ {
		w := doRequest(router, "POST", "/studio/api/sql", map[string]interface{}{"query": "SELECT 1"})
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("no limit configured, but got 429 on request %d", i)
		}
	}
}

func TestRateLimit_ImportEndpoint(t *testing.T) {
	router, _ := setupHeaderRouter(t, Config{
		RateLimit: RateLimitConfig{ImportPerMinute: 2},
	})
	var got429 bool
	for i := 0; i < 5; i++ {
		rec := doMultipartRequest(router, "/studio/api/import/data", "file", "x.csv",
			"name,email\nA,a@x\n", map[string]string{"table": "test_users"})
		if rec.Code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Error("expected a 429 after exceeding the import rate limit")
	}
}
