# Security

GORM Studio is designed as a **development and debugging tool**. It provides direct access to your database and should be treated accordingly.

## Important Warning

**Do not expose GORM Studio on public-facing servers without authentication and access controls.** By default, anyone who can reach the studio URL has full read/write access to your database.

## Threat Model

GORM Studio is intended for:
- Local development
- Internal staging environments (behind VPN/firewall)
- Admin panels with authentication layers

It is **not** designed for:
- Public-facing production use
- Untrusted user environments
- Multi-tenant applications without isolation

## Built-in Security Measures

### Column Name Validation

All column names used in queries are validated against the introspected schema. The `isValidColumn()` function checks that filter and sort column names actually exist in the target table before they're used in SQL queries. This prevents column name injection.

### Table Name Validation

The `IsValidTable()` function validates table names against the known schema before any query is executed. Requests for non-existent tables return a 404 error.

### Input Filtering

When creating or updating rows, `filterValidColumns()` strips out any fields that don't match valid column names in the schema. Only recognized columns are passed to the database.

### Parameterized Queries

All user-provided values (filter values, search terms, row IDs) are passed as parameterized query arguments (`?`), not interpolated into SQL strings. This prevents SQL injection through values.

### Read-Only Mode

Setting `ReadOnly: true` in the config completely disables all mutation endpoints at the route registration level — they are never registered with Gin, so they return 404:

```go
studio.Mount(router, db, models, studio.Config{
    ReadOnly: true,
})
```

### SQL Editor Disable

Setting `DisableSQL: true` prevents the SQL endpoint from being registered:

```go
studio.Mount(router, db, models, studio.Config{
    DisableSQL: true,
})
```

## Adding Authentication

### Basic Auth (Quick Setup)

The simplest approach for development/staging:

```go
router := gin.Default()

// Create an authorized group
authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
    "admin": "your-secure-password",
}))

// Mount studio under the authorized group
// Note: You'll need to set up routes manually in this case
```

### JWT / Custom Middleware

For production-like environments, use your existing authentication middleware:

```go
func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        // Validate token...
        if !valid {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }
        c.Next()
    }
}

// Apply before studio routes
router.Use(AuthRequired())
studio.Mount(router, db, models)
```

See [Adding JWT Authentication](examples/with-auth.md) for a complete example.

## Recommended Deployment Practices

### Development

- No special precautions needed
- Studio runs on localhost, accessible only to the developer

### Staging / Internal

1. **Use authentication** — At minimum, basic auth; ideally your application's auth system
2. **Enable read-only mode** — Prevent accidental data modification
3. **Disable SQL editor** — Prevent arbitrary query execution
4. **Use HTTPS** — Protect credentials in transit

```go
studio.Mount(router, db, models, studio.Config{
    ReadOnly:   true,
    DisableSQL: true,
})
```

### Production (if you must)

If you need database browsing in production:

1. **Strong authentication** — JWT, OAuth, or SSO
2. **Role-based access** — Only allow authorized administrators
3. **Read-only mode** — Always
4. **Disable SQL** — Always
5. **Network isolation** — Bind to internal interface or use VPN
6. **Audit logging** — Log all access to the studio
7. **Rate limiting** — Prevent abuse

```go
// Production-safe configuration
adminGroup := router.Group("/admin",
    AuthRequired(),
    RoleRequired("database_admin"),
    RateLimiter(),
)

studio.Mount(adminGroup.(*gin.Engine), db, models, studio.Config{
    Prefix:     "/studio",
    ReadOnly:   true,
    DisableSQL: true,
})
```

## Network Isolation

### Bind to Localhost Only

```go
// Only accessible from the local machine
router.Run("127.0.0.1:8080")
```

### Separate Port

Run the studio on a different port from your main application:

```go
// Main app on :8080
go mainRouter.Run(":8080")

// Studio on :9090 (internal only)
studioRouter := gin.Default()
studio.Mount(studioRouter, db, models)
studioRouter.Run("127.0.0.1:9090")
```

### Behind a Reverse Proxy

If using Nginx, restrict access by IP:

```nginx
location /studio {
    allow 10.0.0.0/8;
    allow 192.168.0.0/16;
    deny all;

    proxy_pass http://localhost:8080;
}
```

## Known Security Considerations

### SQL Editor

When the SQL editor is enabled, users can execute **any** SQL query including:
- `DROP TABLE`
- `DELETE FROM` (without WHERE)
- `ALTER TABLE`
- Schema modifications

Always disable the SQL editor in environments where this is unacceptable.

### Raw SQL Injection

While the CRUD endpoints use parameterized queries, the SQL editor endpoint (`POST /api/sql`) executes user-provided SQL directly. This is by design — the SQL editor is a power-user feature for developers.

### Identifier Quoting

Column and table names used in dynamically built SQL are both validated against the schema **and** quoted using dialect-appropriate quoting (`"` for SQLite/PostgreSQL, `` ` `` for MySQL). This provides defense-in-depth: even if a column name somehow bypassed validation, quoting prevents it from breaking out of the identifier context.

### No CSRF Protection

The API does not include CSRF tokens. If the studio is accessible from a web browser with active sessions to other sites, consider adding CSRF middleware.

### No Rate Limiting

There is no built-in rate limiting. For exposed environments, add rate limiting middleware to prevent abuse.
