# Using GORM Studio with PostgreSQL

This example shows how to set up GORM Studio with a PostgreSQL database.

## Prerequisites

- A running PostgreSQL instance
- Go 1.21+

## Installation

```bash
go get github.com/MUKE-coder/gorm-studio/studio
go get gorm.io/driver/postgres
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/MUKE-coder/gorm-studio/studio"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type Product struct {
    ID          uint    `gorm:"primarykey" json:"id"`
    Name        string  `gorm:"size:200;not null" json:"name"`
    Description string  `gorm:"type:text" json:"description"`
    Price       float64 `gorm:"not null" json:"price"`
    InStock     bool    `gorm:"default:true" json:"in_stock"`
    CategoryID  uint    `gorm:"index" json:"category_id"`
    Category    Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

type Category struct {
    ID       uint      `gorm:"primarykey" json:"id"`
    Name     string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
    Products []Product `gorm:"foreignKey:CategoryID" json:"products,omitempty"`
}

func main() {
    dsn := "host=localhost user=postgres password=postgres dbname=myapp port=5432 sslmode=disable"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to PostgreSQL:", err)
    }

    // Auto-migrate
    db.AutoMigrate(&Category{}, &Product{})

    // Create Gin router
    router := gin.Default()

    // Mount GORM Studio
    models := []interface{}{&Product{}, &Category{}}
    err = studio.Mount(router, db, models, studio.Config{
        Prefix: "/studio",
    })
    if err != nil {
        log.Fatal("Failed to mount studio:", err)
    }

    fmt.Println("GORM Studio running at http://localhost:8080/studio")
    router.Run(":8080")
}
```

## PostgreSQL-Specific Notes

### Schema Introspection

GORM Studio queries `information_schema.tables` and `information_schema.columns` to discover the database structure. It automatically detects:

- Table names (from the `public` schema)
- Column names, data types, nullability, and defaults
- Column types use PostgreSQL native types (`integer`, `text`, `character varying`, `boolean`, `timestamp with time zone`, etc.)

### Connection String

Use the standard PostgreSQL DSN format:

```go
dsn := "host=localhost user=postgres password=postgres dbname=myapp port=5432 sslmode=disable"
```

Or use a connection URL:

```go
dsn := "postgresql://postgres:postgres@localhost:5432/myapp?sslmode=disable"
```

### Environment Variables

For production, use environment variables:

```go
import "os"

dsn := fmt.Sprintf(
    "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
    os.Getenv("DB_HOST"),
    os.Getenv("DB_USER"),
    os.Getenv("DB_PASSWORD"),
    os.Getenv("DB_NAME"),
    os.Getenv("DB_PORT"),
)
```

### SQL Editor

When using the SQL editor with PostgreSQL, you can use PostgreSQL-specific SQL:

```sql
-- PostgreSQL-specific queries work in the SQL editor
SELECT * FROM products WHERE price > 100 ORDER BY price DESC;
EXPLAIN ANALYZE SELECT * FROM products WHERE category_id = 1;
SELECT pg_size_pretty(pg_total_relation_size('products'));
```

### Known Considerations

- PostgreSQL schemas other than `public` are not currently supported for introspection
- Array types (`text[]`, `integer[]`) appear as their base type in the column info
- JSONB columns are supported — data displays as JSON strings in the UI
- UUID primary keys work, but the URL parameter is passed as a string
