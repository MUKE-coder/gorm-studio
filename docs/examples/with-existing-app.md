# Adding GORM Studio to an Existing Application

This guide shows how to integrate GORM Studio into an existing Gin + GORM application with minimal changes.

## Typical Existing Application

Suppose you have an existing application like this:

```go
package main

import (
    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type User struct {
    ID       uint   `gorm:"primarykey"`
    Username string `gorm:"size:100;uniqueIndex"`
    Email    string `gorm:"size:200"`
    Password string `gorm:"size:200"`
}

type Order struct {
    ID     uint    `gorm:"primarykey"`
    UserID uint    `gorm:"index"`
    User   User    `gorm:"foreignKey:UserID"`
    Total  float64 `gorm:"type:decimal(10,2)"`
    Status string  `gorm:"size:50;default:pending"`
}

type OrderItem struct {
    ID        uint    `gorm:"primarykey"`
    OrderID   uint    `gorm:"index"`
    Order     Order   `gorm:"foreignKey:OrderID"`
    Product   string  `gorm:"size:200"`
    Quantity  int     `gorm:"not null"`
    UnitPrice float64 `gorm:"type:decimal(10,2)"`
}

func main() {
    db, _ := gorm.Open(postgres.Open("host=localhost dbname=myapp ..."), &gorm.Config{})
    db.AutoMigrate(&User{}, &Order{}, &OrderItem{})

    router := gin.Default()

    // Your existing API routes
    api := router.Group("/api/v1")
    {
        api.GET("/users", listUsers(db))
        api.POST("/orders", createOrder(db))
        // ... more routes
    }

    router.Run(":8080")
}
```

## Step 1: Add the Dependency

```bash
go get github.com/MUKE-coder/gorm-studio/studio
```

## Step 2: Import and Mount

Add just three things:

1. Import the studio package
2. Create a models slice
3. Call `studio.Mount()`

```go
package main

import (
    "github.com/MUKE-coder/gorm-studio/studio"  // Add this import

    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

// ... your existing models stay the same ...

func main() {
    db, _ := gorm.Open(postgres.Open("host=localhost dbname=myapp ..."), &gorm.Config{})
    db.AutoMigrate(&User{}, &Order{}, &OrderItem{})

    router := gin.Default()

    // Your existing API routes — unchanged
    api := router.Group("/api/v1")
    {
        api.GET("/users", listUsers(db))
        api.POST("/orders", createOrder(db))
    }

    // Add GORM Studio — just these lines!
    studio.Mount(router, db, []interface{}{
        &User{},
        &Order{},
        &OrderItem{},
    })

    router.Run(":8080")
}
```

That's it. Your existing routes continue to work, and GORM Studio is available at `/studio`.

## Step 3 (Optional): Development-Only Setup

Use a build tag or environment variable to only include the studio in development:

### Using Environment Variables

```go
import "os"

func main() {
    // ... existing setup ...

    router := gin.Default()
    // ... existing routes ...

    // Only mount studio in development
    if os.Getenv("ENABLE_STUDIO") == "true" {
        studio.Mount(router, db, []interface{}{
            &User{},
            &Order{},
            &OrderItem{},
        })
    }

    router.Run(":8080")
}
```

Run with:
```bash
ENABLE_STUDIO=true go run main.go
```

### Using Build Tags

Create a file `studio_dev.go`:

```go
//go:build dev

package main

import (
    "github.com/MUKE-coder/gorm-studio/studio"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func mountStudio(router *gin.Engine, db *gorm.DB) {
    studio.Mount(router, db, []interface{}{
        &User{},
        &Order{},
        &OrderItem{},
    })
}
```

And a no-op for production `studio_prod.go`:

```go
//go:build !dev

package main

import (
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func mountStudio(router *gin.Engine, db *gorm.DB) {
    // Studio not included in production builds
}
```

Then in `main.go`:

```go
func main() {
    // ... setup ...
    mountStudio(router, db)
    router.Run(":8080")
}
```

Build for development:
```bash
go run -tags dev main.go studio_dev.go
```

Build for production:
```bash
go build -o myapp main.go studio_prod.go
```

## Using a Custom Prefix

If `/studio` conflicts with your existing routes, use a different prefix:

```go
studio.Mount(router, db, models, studio.Config{
    Prefix: "/admin/database",
})
// Available at http://localhost:8080/admin/database
```

## Choosing Which Models to Expose

You don't have to expose all your models. Select only the ones you want visible:

```go
// Only expose Order and OrderItem, not User (which has passwords)
studio.Mount(router, db, []interface{}{
    &Order{},
    &OrderItem{},
})
```

Note: Tables without models will still be discovered via database introspection, but without Go type information or relationship metadata.

## Protecting Sensitive Data

If your models contain sensitive fields (passwords, tokens, etc.), consider using read-only mode:

```go
studio.Mount(router, db, []interface{}{
    &User{},
    &Order{},
    &OrderItem{},
}, studio.Config{
    ReadOnly:   true,   // No writes
    DisableSQL: true,   // No raw SQL
})
```

**Note:** Even in read-only mode, all column data is visible. If you have sensitive columns that shouldn't be viewable, consider not including that model in the models slice.
