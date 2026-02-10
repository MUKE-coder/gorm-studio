# Configuration

GORM Studio is configured via the `studio.Config` struct passed to `studio.Mount()`.

## Config Struct

```go
type Config struct {
    // Prefix is the URL prefix where the studio UI and API are mounted.
    // Default: "/studio"
    Prefix string

    // ReadOnly disables all write operations (create, update, delete, bulk delete).
    // The SQL editor is still available unless DisableSQL is also set.
    // Default: false
    ReadOnly bool

    // DisableSQL hides the SQL editor and disables the POST /api/sql endpoint.
    // Default: false
    DisableSQL bool
}
```

## Default Configuration

If you call `Mount()` without a config, or with an empty config, the defaults are used:

```go
// These two calls are equivalent:
studio.Mount(router, db, models)
studio.Mount(router, db, models, studio.Config{
    Prefix:     "/studio",
    ReadOnly:   false,
    DisableSQL: false,
})
```

## Configuration Options

### Prefix

The URL prefix where GORM Studio is mounted. This affects both the web UI and all API endpoints.

```go
studio.Mount(router, db, models, studio.Config{
    Prefix: "/admin/db",
})
// UI available at:  http://localhost:8080/admin/db
// API available at: http://localhost:8080/admin/db/api/*
```

The prefix must start with `/`. If an empty string is provided, it defaults to `"/studio"`.

### Read-Only Mode

When `ReadOnly` is set to `true`, all mutation endpoints are disabled:

- `POST /api/tables/:table/rows` (create) — not registered
- `PUT /api/tables/:table/rows/:id` (update) — not registered
- `DELETE /api/tables/:table/rows/:id` (delete) — not registered
- `POST /api/tables/:table/rows/bulk-delete` — not registered

The frontend automatically hides the "Add Record" button, edit/delete actions, and row checkboxes.

```go
studio.Mount(router, db, models, studio.Config{
    ReadOnly: true,
})
```

**Note:** The SQL editor can still execute write queries (INSERT, UPDATE, DELETE) unless `DisableSQL` is also set. For a fully read-only setup, enable both:

```go
studio.Mount(router, db, models, studio.Config{
    ReadOnly:   true,
    DisableSQL: true,
})
```

### Disable SQL Editor

When `DisableSQL` is set to `true`:

- The `POST /api/sql` endpoint is not registered
- The "SQL" tab is hidden in the frontend sidebar

```go
studio.Mount(router, db, models, studio.Config{
    DisableSQL: true,
})
```

This is useful for environments where you want to allow record browsing and editing but prevent arbitrary SQL execution.

## Passing Models

The `models` parameter is a slice of pointers to your GORM model structs. GORM Studio uses these for schema introspection via reflection:

```go
models := []interface{}{
    &User{},
    &Profile{},
    &Post{},
    &Comment{},
    &Tag{},
}

studio.Mount(router, db, models, studio.Config{})
```

**Important notes:**

- Always pass **pointers** to model structs (`&User{}`, not `User{}`)
- Include all models you want visible in the studio
- Tables that exist in the database but don't have a corresponding model will still be discovered via direct database introspection, but with less type information
- Models that haven't been migrated yet (no corresponding table) will appear in the schema but show 0 rows

## Config Endpoint

The studio exposes a `GET /api/config` endpoint that returns the current configuration state:

```json
{
  "read_only": false,
  "disable_sql": false,
  "prefix": "/studio"
}
```

The frontend uses this to dynamically show or hide UI elements.

## Example Configurations

### Development (full access)

```go
studio.Mount(router, db, models, studio.Config{
    Prefix: "/studio",
})
```

### Staging (read-only browsing)

```go
studio.Mount(router, db, models, studio.Config{
    Prefix:     "/studio",
    ReadOnly:   true,
    DisableSQL: true,
})
```

### Custom path with SQL only for trusted users

```go
// Public read-only view
studio.Mount(router, db, models, studio.Config{
    Prefix:     "/data-viewer",
    ReadOnly:   true,
    DisableSQL: true,
})
```

## Adding Authentication

GORM Studio does not include built-in authentication, but you can protect it using Gin's middleware system. See the [Security](security.md) guide and [Adding JWT Authentication](examples/with-auth.md) example for details.

```go
// Quick example with basic auth
authorized := router.Group("/studio", gin.BasicAuth(gin.Accounts{
    "admin": "secret-password",
}))
// Note: You'll need to manually register the studio routes on this group
// instead of using studio.Mount() directly. See the auth example for details.
```
