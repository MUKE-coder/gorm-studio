# Setting Up a Read-Only Data Viewer

This example shows how to configure GORM Studio as a safe, read-only data browser for teams that need to view database contents without the ability to modify data.

## Use Cases

- Giving support teams visibility into production data
- Providing QA engineers access to staging databases
- Creating a data dashboard for non-technical stakeholders
- Auditing data without risk of accidental modification

## Basic Read-Only Setup

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

// Your application models
type Customer struct {
    ID        uint   `gorm:"primarykey" json:"id"`
    Name      string `gorm:"size:200" json:"name"`
    Email     string `gorm:"size:200" json:"email"`
    Plan      string `gorm:"size:50" json:"plan"`
    CreatedAt string `json:"created_at"`
}

type Subscription struct {
    ID         uint   `gorm:"primarykey" json:"id"`
    CustomerID uint   `gorm:"index" json:"customer_id"`
    Plan       string `gorm:"size:50" json:"plan"`
    Status     string `gorm:"size:50" json:"status"`
    ExpiresAt  string `json:"expires_at"`
}

type Invoice struct {
    ID         uint    `gorm:"primarykey" json:"id"`
    CustomerID uint    `gorm:"index" json:"customer_id"`
    Amount     float64 `gorm:"type:decimal(10,2)" json:"amount"`
    Status     string  `gorm:"size:50" json:"status"`
    PaidAt     *string `json:"paid_at"`
}

func main() {
    dsn := "host=production-db.internal user=readonly password=... dbname=app sslmode=require"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }

    router := gin.Default()

    models := []interface{}{
        &Customer{},
        &Subscription{},
        &Invoice{},
    }

    // Read-only: no mutations, no SQL editor
    err = studio.Mount(router, db, models, studio.Config{
        Prefix:     "/viewer",
        ReadOnly:   true,
        DisableSQL: true,
    })
    if err != nil {
        log.Fatal("Failed to mount studio:", err)
    }

    fmt.Println("Data viewer running at http://localhost:8080/viewer")
    router.Run(":8080")
}
```

## What Read-Only Mode Disables

When `ReadOnly: true` is set:

| Feature | Status |
|---------|--------|
| Browse tables | Available |
| Search and filter | Available |
| Sort columns | Available |
| View relationships | Available |
| Pagination | Available |
| Create records | **Disabled** |
| Edit records | **Disabled** |
| Delete records | **Disabled** |
| Bulk delete | **Disabled** |
| Checkboxes | **Hidden** |
| Action buttons | **Hidden** |
| "Add Record" button | **Hidden** |

When `DisableSQL: true` is also set:

| Feature | Status |
|---------|--------|
| SQL editor tab | **Hidden** |
| SQL API endpoint | **Not registered** |

## Using a Read-Only Database User

For maximum safety, connect with a database user that only has read permissions:

### PostgreSQL

```sql
-- Create a read-only user
CREATE ROLE studio_reader WITH LOGIN PASSWORD 'secure-password';
GRANT CONNECT ON DATABASE myapp TO studio_reader;
GRANT USAGE ON SCHEMA public TO studio_reader;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO studio_reader;

-- For future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO studio_reader;
```

```go
dsn := "host=localhost user=studio_reader password=secure-password dbname=myapp sslmode=disable"
```

### MySQL

```sql
-- Create a read-only user
CREATE USER 'studio_reader'@'%' IDENTIFIED BY 'secure-password';
GRANT SELECT ON myapp.* TO 'studio_reader'@'%';
FLUSH PRIVILEGES;
```

```go
dsn := "studio_reader:secure-password@tcp(127.0.0.1:3306)/myapp?parseTime=True"
```

This provides defense-in-depth: even if the `ReadOnly` flag were bypassed, the database user can't perform writes.

## Adding Basic Auth for Team Access

```go
router := gin.Default()

// Protect the viewer with basic auth
authedRouter := gin.Default()
authedRouter.Use(gin.BasicAuth(gin.Accounts{
    "support":     "support-password",
    "qa":          "qa-password",
    "engineering": "eng-password",
}))

studio.Mount(authedRouter, db, models, studio.Config{
    Prefix:     "/viewer",
    ReadOnly:   true,
    DisableSQL: true,
})

authedRouter.Run(":8080")
```

## Running on a Separate Port

Keep the data viewer isolated from your main application:

```go
func main() {
    db := connectToDatabase()

    // Main application on :8080
    mainRouter := gin.Default()
    setupMainRoutes(mainRouter, db)
    go mainRouter.Run(":8080")

    // Read-only viewer on :9090 (internal network only)
    viewerRouter := gin.Default()
    studio.Mount(viewerRouter, db, models, studio.Config{
        Prefix:     "/viewer",
        ReadOnly:   true,
        DisableSQL: true,
    })

    // Bind to internal interface only
    viewerRouter.Run("10.0.0.1:9090")
}
```

## Selective Table Exposure

Only expose tables that should be visible. Exclude tables with sensitive data:

```go
// Don't include User model (has passwords) or APIKey model
models := []interface{}{
    &Customer{},
    &Subscription{},
    &Invoice{},
    // &User{} — excluded: contains password hashes
    // &APIKey{} — excluded: contains sensitive tokens
}
```

Note that tables without models can still be discovered via database introspection. For complete table hiding, use a database user that only has SELECT permissions on the specific tables you want to expose.
