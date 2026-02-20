package studio

import (
	"fmt"
	"log"
	"net/http"

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
	// When set, all studio routes (UI and API) are protected by this middleware.
	AuthMiddleware gin.HandlerFunc
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

	handlers, err := NewHandlers(db, models)
	if err != nil {
		return fmt.Errorf("mounting studio: %w", err)
	}
	handlers.ReadOnly = cfg.ReadOnly

	group := router.Group(cfg.Prefix)

	// Add CORS middleware if configured
	if len(cfg.CORSAllowOrigins) > 0 {
		group.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.CORSAllowOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
		}))
	}

	// Add auth middleware if configured
	if cfg.AuthMiddleware != nil {
		group.Use(cfg.AuthMiddleware)
	}

	{
		// Serve frontend
		group.GET("", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, GetFrontendHTML(cfg))
		})

		// API routes
		api := group.Group("/api")
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

			// Import (gated by ReadOnly)
			if !cfg.ReadOnly {
				api.POST("/import/schema", handlers.ImportSchema)
				api.POST("/import/data", handlers.ImportData)
				api.POST("/import/models", handlers.ImportGoModels)
			}

			// Raw SQL
			if !cfg.DisableSQL {
				api.POST("/sql", handlers.ExecuteSQL)
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
