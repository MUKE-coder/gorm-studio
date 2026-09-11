# Security

GORM Studio is designed as a **development and debugging tool**. It provides direct access to your database and should be treated accordingly.

## Important Warning

**Do not expose GORM Studio on public-facing servers without authentication and access controls.** By default, anyone who can reach the studio URL has full read/write access to your database.

## ⚠️ Studio bypasses application-layer access control

GORM Studio talks to your database through GORM but **outside your application's
request handlers**. Any access control your app enforces in those handlers —
GORM callbacks, query scopes, multi-tenancy filters, row-ownership checks
(e.g. Grit's `--tenant-owned` / `--owned-by`), soft ACLs — **does not apply to
Studio**. An operator with Studio access can read and edit **every row in every
table**, across all tenants and owners, unless you tell Studio how to restrict
them.

If you mount Studio in a multi-tenant or row-scoped application, you must do one
of the following:

1. **Set a `Scope`** (see [Row-level scoping](#row-level-scoping-multi-tenancy))
   so every query Studio builds is constrained the way your app would constrain
   it, **and** set `DisableSQL: true` (the raw SQL editor cannot be scoped).
2. **Restrict who can reach Studio** to trusted operators who are allowed to see
   all tenants' data, and treat that access as god-mode.

Do not assume "authenticated Studio access" is equivalent to "application-level
access controls apply." It is not.

## Threat Model

GORM Studio is intended for:
- Local development
- Internal staging environments (behind VPN/firewall)
- Admin panels with authentication layers

It is **not** designed for:
- Public-facing production use
- Untrusted user environments
- Multi-tenant applications **without a configured `Scope`** (see
  [Row-level scoping](#row-level-scoping-multi-tenancy)) — without it, Studio
  sees every tenant's data

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

Setting `ReadOnly: true` in the config completely disables all mutation endpoints at the route registration level — they are never registered with Gin, so they return 404. The SQL editor, which stays registered, also enforces read-only mode: only genuine read statements (`SELECT`/`EXPLAIN`/`SHOW`/`DESCRIBE` and read-only `PRAGMA`) are allowed, and any write — including `PRAGMA x = y` or a statement mislabeled as a read — returns 403:

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

## Row-level scoping (multi-tenancy)

Set `Scope` to constrain every query Studio builds — row listing, single-row
reads, updates, deletes, relations, and exports. Use it to reproduce the
row-level isolation your application enforces (tenant, owner, org, …).

```go
studio.Mount(router, db, models, studio.Config{
    DisableSQL: true, // required: the raw SQL editor cannot be scoped
    Scope: func(c *gin.Context, table string, tx *gorm.DB) *gorm.DB {
        // Resolve the active tenant from the request (set by your auth middleware).
        tenantID := c.GetString("tenant_id")
        // Only constrain tables that actually have the column.
        switch table {
        case "orders", "invoices", "customers":
            return tx.Where("tenant_id = ?", tenantID)
        }
        return tx // other tables unaffected
    },
})
```

Notes and limitations:

- **The SQL editor is not scoped.** Always pair `Scope` with `DisableSQL: true`.
  Studio logs a warning at startup if you don't.
- **Row creation is not auto-scoped.** A `Scope` is a `WHERE` constraint, so it
  governs which rows can be read/updated/deleted, not what a new row is stamped
  with. If operators must not create cross-tenant rows, mark those tables
  read-only (below) or keep Studio read-only.
- Return the query unchanged for tables the scope doesn't apply to.

## Per-table permissions

`TablePolicy` restricts individual tables independently of the global
`ReadOnly` flag:

```go
studio.Mount(router, db, models, studio.Config{
    TablePolicy: studio.TablePolicy{
        Hidden:   []string{"secrets", "payment_tokens"}, // never exposed at all
        ReadOnly: []string{"audit_log", "ledger_entries"}, // browsable, not editable
    },
})
```

- **Hidden** tables are omitted from the schema, return 404 on direct access,
  and are excluded from all exports. References to them are scrubbed from other
  tables' relations and foreign keys so their names don't leak.
- **ReadOnly** tables can be browsed and exported, but create/update/delete,
  bulk delete, and imports targeting them return 403.

## Audit logging

Set `AuditLogger` to record every mutation performed through Studio (row
create/update/delete, bulk delete, raw SQL writes, and imports). Use
`studio.DefaultAuditLogger` for simple stdout logging, or supply your own to
forward events to your logging/audit pipeline.

```go
studio.Mount(router, db, models, studio.Config{
    AuditLogger: func(e studio.AuditEvent) {
        // e.Time, e.Actor, e.Action, e.Table, e.RowID, e.Rows, e.Query, e.Success, e.Err
        myAuditSink.Record(e)
    },
})
```

The `Actor` is taken from the request context if your auth middleware records
it under `studio_user`, `user`, `username`, or gin's basic-auth user key.

## Import safety

Studio's importers accept several hand-parsed formats (SQL, JSON, YAML, DBML,
CSV, XLSX, and Go source). To bound their blast radius:

- **Size limit.** Uploads larger than `MaxImportBytes` (default 32 MiB) are
  rejected with 413 before the body is buffered. Set a negative value to
  disable.
- **Row limit.** A single import may insert at most `MaxImportRows` rows
  (default 100,000). SQL data imports are also capped by statement count.
- **Streaming XLSX.** Excel files are read row-by-row so a small upload that
  decompresses to a huge sheet can't exhaust memory.
- **Column-type validation.** Imported column types are checked against a plain
  type-name pattern before being written into `CREATE TABLE`, so a crafted type
  string can't inject DDL.
- **INSERT-only data SQL.** SQL data imports run inside a transaction and
  execute only `INSERT` statements; any other statement aborts the whole import
  with nothing applied.

```go
studio.Mount(router, db, models, studio.Config{
    MaxImportBytes: 8 << 20, // 8 MiB
    MaxImportRows:  10000,
})
```

### Dry run

Add `?dry_run=true` to any import endpoint (`/api/import/schema`,
`/api/import/data`, `/api/import/models`) to preview what it would do — tables
that would be created, rows that would be inserted — without applying any
change.

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

The SQL editor accepts a **single** statement per request. Before execution the
query has its comments stripped and is checked against a keyword blocklist, so
the following are rejected (403) regardless of comments, leading whitespace, or
letter case:

- `DROP`, `ALTER`, `TRUNCATE`, `CREATE`
- `ATTACH`, `DETACH`
- `GRANT`, `REVOKE`
- `VACUUM`, `REINDEX` (e.g. `VACUUM INTO '<path>'` would write an arbitrary file)

Multiple statements separated by `;` are rejected (400) so a benign-looking
`SELECT` cannot smuggle a trailing write.

The editor can still run non-DDL writes (`INSERT`/`UPDATE`/`DELETE`) when
`ReadOnly` is not set — including a `DELETE`/`UPDATE` without a `WHERE` clause.
Disable the SQL editor (`DisableSQL: true`) in environments where arbitrary
DML is unacceptable, and always combine it with `ReadOnly: true` when browsing
production data.

### Raw SQL Injection

While the CRUD endpoints use parameterized queries, the SQL editor endpoint (`POST /api/sql`) executes user-provided SQL directly. This is by design — the SQL editor is a power-user feature for developers.

### Identifier Quoting

Column and table names used in dynamically built SQL are both validated against the schema **and** quoted using dialect-appropriate quoting (`"` for SQLite/PostgreSQL, `` ` `` for MySQL). This provides defense-in-depth: even if a column name somehow bypassed validation, quoting prevents it from breaking out of the identifier context.

### No CSRF Protection

The API does not include CSRF tokens. If the studio is accessible from a web browser with active sessions to other sites, consider adding CSRF middleware.

### No Rate Limiting

There is no built-in rate limiting. For exposed environments, add rate limiting middleware to prevent abuse.
