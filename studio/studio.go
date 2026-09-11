package studio

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Config holds configuration for the studio
type Config struct {
	// Prefix is the URL prefix for the studio (default: "/studio")
	Prefix string
	// ReadOnly disables write operations
	ReadOnly bool
	// DisableSQL disables the raw SQL editor
	DisableSQL bool
	// CORSAllowOrigins is a list of allowed origins for CORS. If empty, CORS middleware is not added.
	CORSAllowOrigins []string
	// AuthMiddleware is an optional Gin middleware function for authentication.
	// When set, all studio API routes are protected by this middleware.
	// The frontend HTML is served without auth; the React UI shows a login form on 401.
	AuthMiddleware gin.HandlerFunc

	// Scope, when set, is applied to every data query Studio builds (row
	// listing/reading, updates, deletes, relations, and exports). Use it to
	// enforce row-level constraints — most importantly tenant isolation — that
	// your application normally applies through GORM callbacks but that Studio
	// bypasses because it talks to the database directly.
	//
	// The function receives the request context (so it can read the active
	// tenant/user), the target table name, and the query being built, and
	// should return the query with any additional constraints applied. Return
	// the query unchanged for tables the scope does not apply to.
	//
	// IMPORTANT: the raw SQL editor cannot be scoped. When Scope is set you
	// should also set DisableSQL: true; otherwise an operator can read/write
	// across the boundary the Scope enforces.
	Scope func(c *gin.Context, table string, tx *gorm.DB) *gorm.DB

	// TablePolicy controls per-table visibility and mutability, independent of
	// the global ReadOnly flag.
	TablePolicy TablePolicy

	// AuditLogger, when set, receives an event for every mutating action
	// performed through Studio (row create/update/delete, bulk delete, raw SQL
	// writes, and imports). Use DefaultAuditLogger for simple stdout logging.
	AuditLogger func(AuditEvent)

	// MaxImportBytes caps the size of an uploaded import file. Requests larger
	// than this are rejected before the body is read, bounding memory use
	// against decompression/SQL bombs. Zero uses DefaultMaxImportBytes; a
	// negative value disables the limit.
	MaxImportBytes int64

	// MaxImportRows caps how many rows a single import may insert (per file).
	// Zero uses DefaultMaxImportRows; a negative value disables the limit.
	MaxImportRows int

	// RateLimit applies Studio-specific per-client-IP rate limiting to the SQL
	// and import endpoints. The zero value applies no limit.
	RateLimit RateLimitConfig

	// ImportTimeout bounds how long a single import may run. Zero uses
	// DefaultImportTimeout; a negative value disables the timeout.
	ImportTimeout time.Duration
}

// contentSecurityPolicy is served with the Studio HTML page. It pins script and
// style origins to the CDNs the app loads from while still permitting the
// inline/eval the bundled Babel+React setup requires, and forbids framing.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' https://cdnjs.cloudflare.com 'unsafe-inline' 'unsafe-eval'; " +
	"style-src 'self' https://fonts.googleapis.com 'unsafe-inline'; " +
	"font-src https://fonts.gstatic.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'"

// securityHeaders sets conservative security headers on every Studio response.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// Import limit defaults.
const (
	// DefaultMaxImportBytes is the default cap on uploaded import file size (32 MiB).
	DefaultMaxImportBytes int64 = 32 << 20
	// DefaultMaxImportRows is the default cap on rows inserted per import.
	DefaultMaxImportRows = 100_000
	// DefaultImportTimeout is the default time bound on a single import.
	DefaultImportTimeout = 30 * time.Second
)

// TablePolicy restricts which tables Studio exposes and which it may mutate.
type TablePolicy struct {
	// Hidden tables are never exposed: they are omitted from the schema and any
	// direct request for them returns 404. Use this for tables holding secrets
	// (encrypted PII, payment tokens, credentials).
	Hidden []string
	// ReadOnly tables can be browsed and exported but never mutated through
	// Studio; write requests return 403. The global ReadOnly flag still wins
	// over this (it disables all mutation routes entirely).
	ReadOnly []string
}

// DefaultConfig returns the default studio configuration
func DefaultConfig() Config {
	return Config{
		Prefix:     "/studio",
		ReadOnly:   false,
		DisableSQL: false,
	}
}

// Mount registers the studio routes on a Gin engine
func Mount(router *gin.Engine, db *gorm.DB, models []interface{}, configs ...Config) error {
	cfg := DefaultConfig()
	if len(configs) > 0 {
		cfg = configs[0]
		if cfg.Prefix == "" {
			cfg.Prefix = "/studio"
		}
	}

	// Warn if no auth middleware is configured
	if cfg.AuthMiddleware == nil {
		log.Println("[GORM Studio] WARNING: No authentication middleware configured. Studio routes are publicly accessible. Add AuthMiddleware to protect your data.")
	}
	if !cfg.ReadOnly && cfg.AuthMiddleware == nil {
		log.Println("[GORM Studio] WARNING: Write operations are enabled without authentication. Consider setting ReadOnly: true or adding AuthMiddleware.")
	}
	if !cfg.DisableSQL && cfg.AuthMiddleware == nil {
		log.Println("[GORM Studio] WARNING: Raw SQL endpoint is enabled without authentication. Consider setting DisableSQL: true or adding AuthMiddleware.")
	}
	if cfg.Scope != nil && !cfg.DisableSQL {
		log.Println("[GORM Studio] WARNING: A Scope is configured but the raw SQL editor is enabled. The SQL editor bypasses Scope — set DisableSQL: true to keep row-level isolation (e.g. multi-tenancy) enforced.")
	}

	handlers, err := NewHandlers(db, models)
	if err != nil {
		return fmt.Errorf("mounting studio: %w", err)
	}
	handlers.ReadOnly = cfg.ReadOnly
	handlers.Scope = cfg.Scope
	handlers.Audit = cfg.AuditLogger
	handlers.Hidden = newNameSet(cfg.TablePolicy.Hidden)
	handlers.ReadOnlyTables = newNameSet(cfg.TablePolicy.ReadOnly)

	handlers.MaxImportBytes = cfg.MaxImportBytes
	if handlers.MaxImportBytes == 0 {
		handlers.MaxImportBytes = DefaultMaxImportBytes
	}
	handlers.MaxImportRows = cfg.MaxImportRows
	if handlers.MaxImportRows == 0 {
		handlers.MaxImportRows = DefaultMaxImportRows
	}
	handlers.ImportTimeout = cfg.ImportTimeout
	if handlers.ImportTimeout == 0 {
		handlers.ImportTimeout = DefaultImportTimeout
	}

	sqlLimiter := newRateLimiter(cfg.RateLimit.SQLPerMinute)
	importLimiter := newRateLimiter(cfg.RateLimit.ImportPerMinute)

	group := router.Group(cfg.Prefix)

	// Conservative security headers on every Studio response.
	group.Use(securityHeaders())

	// Add CORS middleware if configured
	if len(cfg.CORSAllowOrigins) > 0 {
		group.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.CORSAllowOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
		}))
	}

	// Serve frontend without auth (React app handles login UI)
	group.GET("", func(c *gin.Context) {
		c.Header("Content-Security-Policy", contentSecurityPolicy)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, GetFrontendHTML(cfg))
	})

	{
		// API routes - protected by auth middleware
		api := group.Group("/api")

		if cfg.AuthMiddleware != nil {
			api.Use(func(c *gin.Context) {
				cfg.AuthMiddleware(c)
				// Strip WWW-Authenticate header to prevent browser native auth popup.
				// gin.BasicAuth sets "WWW-Authenticate: Basic realm=..." which causes
				// browsers to show their native popup instead of our React login page.
				if c.IsAborted() {
					c.Writer.Header().Del("WWW-Authenticate")
				}
			})
		}

		{
			// Schema
			api.GET("/schema", handlers.GetSchema)
			api.POST("/schema/refresh", handlers.RefreshSchema)

			// CRUD
			api.GET("/tables/:table/rows", handlers.GetRows)
			api.GET("/tables/:table/rows/:id", handlers.GetRow)

			if !cfg.ReadOnly {
				api.POST("/tables/:table/rows", handlers.CreateRow)
				api.PUT("/tables/:table/rows/:id", handlers.UpdateRow)
				api.DELETE("/tables/:table/rows/:id", handlers.DeleteRow)
				api.POST("/tables/:table/rows/bulk-delete", handlers.BulkDelete)
			}

			// Relations
			api.GET("/tables/:table/rows/:id/relations/:relation", handlers.GetRelatedRows)

			// Export (per-table)
			api.GET("/tables/:table/export", handlers.ExportTable)

			// Export (full database)
			api.GET("/export/schema", handlers.ExportSchema)
			api.GET("/export/data", handlers.ExportAllData)
			api.GET("/export/models", handlers.ExportGoModels)

			// Import (gated by ReadOnly, optionally rate-limited)
			if !cfg.ReadOnly {
				api.POST("/import/schema", withRateLimit(importLimiter, handlers.ImportSchema)...)
				api.POST("/import/data", withRateLimit(importLimiter, handlers.ImportData)...)
				api.POST("/import/models", withRateLimit(importLimiter, handlers.ImportGoModels)...)
			}

			// Raw SQL (optionally rate-limited)
			if !cfg.DisableSQL {
				api.POST("/sql", withRateLimit(sqlLimiter, handlers.ExecuteSQL)...)
			}

			// DB stats
			api.GET("/stats", handlers.GetDBStats)

			// Config info
			api.GET("/config", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"read_only":   cfg.ReadOnly,
					"disable_sql": cfg.DisableSQL,
					"prefix":      cfg.Prefix,
				})
			})
		}
	}

	return nil
}
