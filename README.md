# 🗄️ GORM Studio

A **Prisma Studio-like** visual database browser and editor for Go applications using GORM. Browse, filter, create, edit, and delete records through a sleek web UI.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![GORM](https://img.shields.io/badge/GORM-v2-FF6B6B)
![License](https://img.shields.io/badge/license-MIT-green)

## ✨ Features

- **Schema Discovery** — Introspects your database AND parses GORM model structs via reflection
- **Browse & Filter** — Paginated data grid with column sorting and full-text search
- **CRUD Operations** — Create, edit, and delete records through modal forms
- **Relationship Navigation** — See and navigate foreign key relationships between tables
- **Raw SQL Editor** — Execute any SQL query with syntax highlighting and history
- **Bulk Operations** — Select multiple rows for batch deletion
- **Zero Config** — Just pass your `*gorm.DB` and model list — one line to mount

## 🚀 Quick Start

### 1. Install

```bash
# In your Go project
go get github.com/MUKE-coder/gorm-studio/studio
```

Or copy the `studio/` package into your project.

### 2. Mount in your Gin app

```go
package main

import (
	"log"

	"github.com/MUKE-coder/gorm-studio/studio"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID    uint   `gorm:"primarykey"`
	Name  string `gorm:"size:100"`
	Email string `gorm:"size:200;uniqueIndex"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	db.AutoMigrate(&User{})

	router := gin.Default()

	// Mount GORM Studio — that's it!
	studio.Mount(router, db, []interface{}{&User{}})

	router.Run(":8080")
}

```

### 3. Open in browser

```
http://localhost:8080/studio
```

## 🎛️ Configuration

```go
studio.Mount(router, db, models, studio.Config{
    Prefix:     "/studio",    // URL prefix (default: "/studio")
    ReadOnly:   false,        // Disable write operations
    DisableSQL: false,        // Disable raw SQL editor
})
```

## 📁 Project Structure

```
studio/
├── studio.go      # Mount function — registers routes on Gin
├── schema.go      # Schema introspection (DB + GORM reflection)
├── handlers.go    # REST API handlers (CRUD, filtering, SQL)
└── frontend.go    # Embedded React SPA (served as HTML string)
```

## 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET`  | `/studio` | Web UI |
| `GET`  | `/studio/api/schema` | Get full database schema |
| `POST` | `/studio/api/schema/refresh` | Re-introspect schema |
| `GET`  | `/studio/api/tables/:table/rows` | List rows (paginated, filterable) |
| `GET`  | `/studio/api/tables/:table/rows/:id` | Get single row |
| `POST` | `/studio/api/tables/:table/rows` | Create row |
| `PUT`  | `/studio/api/tables/:table/rows/:id` | Update row |
| `DELETE` | `/studio/api/tables/:table/rows/:id` | Delete row |
| `POST` | `/studio/api/tables/:table/rows/bulk-delete` | Bulk delete |
| `GET`  | `/studio/api/tables/:table/rows/:id/relations/:rel` | Get related rows |
| `POST` | `/studio/api/sql` | Execute raw SQL |

### Query Parameters for listing rows:

- `page` — Page number (default: 1)
- `page_size` — Rows per page (default: 50, max: 500)
- `sort_by` — Column to sort by
- `sort_order` — `asc` or `desc`
- `search` — Full-text search across all text columns
- `filter_<column>` — Filter by column value (use `%` for LIKE)

## 🗃️ Supported Databases

- ✅ SQLite
- ✅ PostgreSQL
- ✅ MySQL

## 🏗️ Running the Demo

```bash
cd gorm-studio
go mod tidy
go run main.go
```

Then open http://localhost:8080/studio — you'll see a demo blog database with Users, Posts, Comments, Tags, and Profiles.

## 📝 Notes

- **Development tool only** — Do not expose in production without authentication
- The SQL editor allows both read and write queries
- Set `ReadOnly: true` to disable all mutations
- Set `DisableSQL: true` to hide the SQL editor
- Schema is cached at startup; use the refresh button to re-introspect

## 🔒 Security Considerations

GORM Studio is designed as a **development/debugging tool**. If you need to use it in staging or production:

1. Add authentication middleware before the studio routes
2. Use `ReadOnly: true` to prevent accidental mutations
3. Use `DisableSQL: true` to prevent arbitrary SQL execution
4. Restrict access via network policies

```go
// Example: protect with basic auth
studioGroup := router.Group("/studio", gin.BasicAuth(gin.Accounts{
    "admin": "secret",
}))
// Then mount studio manually on the group...
```

## License

MIT
