# 🗄️ GORM Studio

A **Prisma Studio-like** visual database browser and editor for Go applications using GORM. Browse schemas, manage data, run SQL, export ERD diagrams, import data, and generate Go models — all from a single `studio.Mount()` call.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![GORM](https://img.shields.io/badge/GORM-v2-FF6B6B)
![License](https://img.shields.io/badge/license-MIT-green)

## ✨ Features

- **Schema Discovery** — Introspects your database AND parses GORM model structs via reflection
- **Browse & Filter** — Paginated data grid with column sorting and full-text search
- **CRUD Operations** — Create, edit, and delete records through modal forms
- **Relationship Navigation** — See and navigate foreign key relationships (has_one, has_many, belongs_to, many_to_many)
- **Raw SQL Editor** — Execute SQL queries with automatic read/write detection and DDL blocking
- **Bulk Operations** — Select multiple rows for batch deletion
- **Schema Export** — Export as SQL DDL, JSON, YAML, DBML, PNG ERD diagram, or PDF ERD diagram
- **Data Export** — Export entire database as JSON, CSV (ZIP), or SQL INSERT statements
- **Data Import** — Import data from JSON, CSV, SQL, or Excel (.xlsx) files
- **Schema Import** — Import schemas from SQL, JSON, YAML, or DBML files to create tables
- **Go Code Generation** — Generate Go model structs from your database schema with proper GORM tags
- **Go Models Import** — Upload a `.go` file with struct definitions to create database tables
- **Authentication** — Built-in `AuthMiddleware` support for protecting routes
- **Security** — DDL blocking, SQL injection prevention, CSV formula injection protection, SRI hashes
- **Zero Config** — Just pass your `*gorm.DB` and model list — one line to mount

## 📖 Documentation

Full documentation is available at the [GORM Studio Docs Site](docs-site/).

To run the docs locally:

```bash
cd docs-site
npm install
npm run dev
```

## 🚀 Quick Start

### 1. Install

```bash
# Create a new Project
go mod init github.com/yourusername/your-project

# Install GORM Studio
go get github.com/MUKE-coder/gorm-studio/studio
```

### Install Other Packages

```bash
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get github.com/glebarez/sqlite       # for SQLite (pure Go, no CGO)
go get gorm.io/driver/postgres           # for PostgreSQL
go get gorm.io/driver/mysql              # for MySQL
```

### 2A. Mount in your Gin app USING SQLITE

```go
package main

import (
	"log"
	"net/http"

	"github.com/MUKE-coder/gorm-studio/studio"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID    uint   `gorm:"primarykey" json:"id"`
	Name  string `gorm:"size:100" json:"name" binding:"required"`
	Email string `gorm:"size:200;uniqueIndex" json:"email" binding:"required,email"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	db.AutoMigrate(&User{})

	router := gin.Default()

	// POST /users - Create a new user
	router.POST("/users", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if result := db.Create(&user); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": user})
	})

	// Mount GORM Studio — that's it!
	studio.Mount(router, db, []interface{}{&User{}})

	router.Run(":8080")
}
```

### 2B. Mount in your Gin app USING PostgreSQL

#### ENV FILE

```env
PGHOST=''
PGDATABASE='neondb'
PGUSER='neondb_owner'
PGPASSWORD=''
PGSSLMODE='require'
PGCHANNELBINDING='require'
```

#### MAIN FILE

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/MUKE-coder/gorm-studio/studio"
	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID    uint   `gorm:"primarykey"`
	Name  string `gorm:"size:100"`
	Email string `gorm:"size:200;uniqueIndex"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s channel_binding=%s",
		os.Getenv("PGHOST"),
		os.Getenv("PGUSER"),
		os.Getenv("PGPASSWORD"),
		os.Getenv("PGDATABASE"),
		os.Getenv("PGSSLMODE"),
		os.Getenv("PGCHANNELBINDING"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
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
    Prefix:           "/studio",       // URL prefix (default: "/studio")
    ReadOnly:         false,           // Disable write operations
    DisableSQL:       false,           // Disable raw SQL editor
    CORSAllowOrigins: []string{},     // Allowed CORS origins
    AuthMiddleware:   nil,             // Authentication middleware
})
```

### Authentication

```go
// Protect with basic auth
studio.Mount(router, db, models, studio.Config{
    AuthMiddleware: gin.BasicAuth(gin.Accounts{
        "admin": "secret-password",
    }),
})

// Or with JWT / custom middleware
studio.Mount(router, db, models, studio.Config{
    AuthMiddleware: func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        // Validate token...
        c.Next()
    },
})
```

### Production Configuration

```go
studio.Mount(router, db, models, studio.Config{
    Prefix:     "/admin/studio",
    ReadOnly:   true,
    DisableSQL: true,
    AuthMiddleware: authMiddleware,
    CORSAllowOrigins: []string{"https://myapp.com"},
})
```

## 📁 Project Structure

```
studio/
├── studio.go          # Mount function — registers routes on Gin
├── schema.go          # Schema introspection (DB + GORM reflection)
├── handlers.go        # REST API handlers (CRUD, filtering, SQL, per-table export)
├── frontend.go        # Embedded React SPA (served as HTML string)
├── codegen.go         # Go struct code generation from schema
├── sql_parser.go      # CREATE TABLE SQL parser (SQLite/Postgres/MySQL)
├── erd.go             # ERD diagram renderer (PNG + PDF)
├── export_schema.go   # Schema export (SQL, JSON, YAML, DBML, PNG, PDF)
├── export_data.go     # Full database data export (JSON, CSV ZIP, SQL)
├── import_schema.go   # Schema import (SQL, JSON, YAML, DBML)
├── import_data.go     # Data import (JSON, CSV, SQL, Excel)
└── import_models.go   # Go model import (parse structs, create tables)
```

## 🔌 API Endpoints

### Schema & Config

| Method | Endpoint                     | Description              |
| ------ | ---------------------------- | ------------------------ |
| `GET`  | `/studio`                    | Web UI                   |
| `GET`  | `/studio/api/schema`         | Get full database schema |
| `POST` | `/studio/api/schema/refresh` | Re-introspect schema     |
| `GET`  | `/studio/api/config`         | Get current config       |
| `GET`  | `/studio/api/stats`          | DB connection pool stats |

### CRUD Operations

| Method   | Endpoint                                            | Description                       |
| -------- | --------------------------------------------------- | --------------------------------- |
| `GET`    | `/studio/api/tables/:table/rows`                    | List rows (paginated, filterable) |
| `GET`    | `/studio/api/tables/:table/rows/:id`                | Get single row                    |
| `POST`   | `/studio/api/tables/:table/rows`                    | Create row                        |
| `PUT`    | `/studio/api/tables/:table/rows/:id`                | Update row                        |
| `DELETE` | `/studio/api/tables/:table/rows/:id`                | Delete row                        |
| `POST`   | `/studio/api/tables/:table/rows/bulk-delete`        | Bulk delete                       |
| `GET`    | `/studio/api/tables/:table/rows/:id/relations/:rel` | Get related rows                  |

### Export

| Method | Endpoint                                   | Description                           |
| ------ | ------------------------------------------ | ------------------------------------- |
| `GET`  | `/studio/api/export/schema?format=<fmt>`   | Export schema (sql/json/yaml/dbml/png/pdf) |
| `GET`  | `/studio/api/export/data?format=<fmt>`     | Export all data (json/csv/sql)        |
| `GET`  | `/studio/api/export/models`                | Download generated Go structs         |
| `GET`  | `/studio/api/tables/:table/export?format=` | Export single table (json/csv)        |

### Import

| Method | Endpoint                        | Description                              |
| ------ | ------------------------------- | ---------------------------------------- |
| `POST` | `/studio/api/import/schema`     | Import schema (.sql/.json/.yaml/.dbml)   |
| `POST` | `/studio/api/import/data`       | Import data (.json/.csv/.sql/.xlsx)      |
| `POST` | `/studio/api/import/models`     | Import Go structs (.go)                  |

### SQL

| Method | Endpoint           | Description      |
| ------ | ------------------ | ---------------- |
| `POST` | `/studio/api/sql`  | Execute raw SQL  |

### Query Parameters for listing rows

- `page` — Page number (default: 1)
- `page_size` — Rows per page (default: 50, max: 500)
- `sort_by` — Column to sort by
- `sort_order` — `asc` or `desc`
- `search` — Full-text search across all text columns
- `filter_<column>` — Filter by column value (use `%` for LIKE)
- `show_deleted` — Include soft-deleted rows (default: false)

## 🗃️ Supported Databases

- ✅ SQLite (via `github.com/glebarez/sqlite` — pure Go, no CGO required)
- ✅ PostgreSQL (via `gorm.io/driver/postgres`)
- ✅ MySQL (via `gorm.io/driver/mysql`)

## 📤 Export Features

### Schema Export
Export your database schema in multiple formats:
- **SQL** — CREATE TABLE DDL statements
- **JSON** — Structured schema definition
- **YAML** — Human-readable schema format
- **DBML** — Database Markup Language (compatible with [dbdiagram.io](https://dbdiagram.io))
- **PNG** — Entity Relationship Diagram as image
- **PDF** — Entity Relationship Diagram as PDF

### Data Export
Export your entire database:
- **JSON** — Single document with all tables and rows
- **CSV** — ZIP archive with one CSV per table (formula injection protected)
- **SQL** — INSERT statements for all data

### Go Models Export
Generate a downloadable `.go` file with struct definitions for all tables, including proper GORM tags, type mapping, and relationship fields.

## 📥 Import Features

### Schema Import
Upload a schema file to create tables:
- `.sql` — CREATE TABLE statements (auto-detects SQLite/PostgreSQL/MySQL dialect)
- `.json` — Structured table definitions
- `.yaml` — YAML table definitions
- `.dbml` — DBML format

Returns created table names and generated Go model code.

### Data Import
Upload data files into existing tables:
- `.json` — Multi-table or single-table format
- `.csv` — Requires `table` query parameter
- `.sql` — Only INSERT statements processed (safe)
- `.xlsx` — Excel files, requires `table` query parameter

### Go Models Import
Upload a `.go` file with struct definitions to automatically create database tables. Parses field types, GORM tags, and creates corresponding columns.

## 🔒 Security

GORM Studio includes multiple security layers:

- **Authentication** — Built-in `AuthMiddleware` support with startup warnings when unprotected
- **Table Validation** — All table names validated against registered models
- **Column Validation** — Only known columns accepted for filtering and sorting
- **Parameterized Queries** — Uses GORM's built-in query parameterization
- **Identifier Quoting** — Dialect-specific quoting (double quotes for SQLite/Postgres, backticks for MySQL)
- **DDL Blocking** — DROP, ALTER, TRUNCATE, CREATE, ATTACH, DETACH, GRANT, REVOKE always blocked in SQL editor
- **CSV Formula Injection** — Cells starting with `=`, `+`, `-`, `@` are prefixed with `'`
- **SRI Hashes** — CDN scripts include Subresource Integrity hashes

### Production Recommendations

1. **Always** add `AuthMiddleware` in non-development environments
2. Use `ReadOnly: true` to prevent accidental mutations
3. Use `DisableSQL: true` to prevent arbitrary SQL execution
4. Restrict access via network policies
5. Use a database user with minimal required permissions

## 🏗️ Running the Demo

```bash
cd gorm-studio
go mod tidy
go run main.go
```

Then open http://localhost:8080/studio — you'll see a demo blog database with Users, Posts, Comments, Tags, and Profiles.

## 📝 Notes

- The SQL editor allows both read and write queries (write can be disabled with `ReadOnly: true`)
- Set `DisableSQL: true` to hide the SQL editor entirely
- Schema is cached at startup; use the refresh button or `POST /api/schema/refresh` to re-introspect
- Import endpoints are only available when `ReadOnly` is false
- Soft-deleted rows (GORM `DeletedAt`) are hidden by default — use `show_deleted=true` to include them

## 🤝 Contributing

Contributions are welcome! Please see the [contributing guide](docs/contributing.md) for details.

## 📄 License

MIT

## 👨‍💻 Author

Built with ❤️ by [JB](https://jb.desishub.com/)

- [YouTube](https://www.youtube.com/@JBWEBDEVELOPER)
- [LinkedIn](https://www.linkedin.com/in/muke-johnbaptist-95bb82198/)
- [Website](https://jb.desishub.com/)
