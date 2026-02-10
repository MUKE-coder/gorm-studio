# Schema Introspection

GORM Studio uses a dual-source schema discovery approach: it combines information from **GORM model reflection** and **direct database introspection** to build a comprehensive picture of your database schema.

## How It Works

When `studio.Mount()` is called, it runs `IntrospectSchema()` which performs these steps:

1. **Parse GORM models** — Uses `gorm.Statement.Parse()` to extract fields, types, relationships, and constraints from your Go struct definitions
2. **Introspect the database** — Queries the database system tables directly (e.g., `sqlite_master`, `information_schema`) to discover tables, columns, and foreign keys
3. **Merge results** — Combines both sources, preferring GORM model information for Go types and relationship details, while filling in gaps with database-level info

```
GORM Models (Go structs)          Database (system tables)
        │                                  │
        ▼                                  ▼
  parseGORMModel()               introspectDatabase()
        │                                  │
        ▼                                  ▼
   Model-based                       DB-based
   TableInfo[]                       TableInfo[]
        │                                  │
        └──────────┬───────────────────────┘
                   ▼
            mergeTableInfo()
                   │
                   ▼
           Final SchemaInfo
```

## GORM Model Reflection

### What It Reads

For each model passed to `Mount()`, the introspector calls `gorm.Statement.Parse()` and extracts:

| Information | Source | Example |
|-------------|--------|---------|
| Table name | `stmt.Schema.Table` | `"users"` |
| Column name | `field.DBName` | `"email"` |
| Data type | `field.DataType` | `"string"` |
| Go type | `field.FieldType.String()` | `"string"` |
| Primary key | `field.PrimaryKey` | `true` |
| Nullable | `!field.NotNull` | `false` |
| Default value | `field.DefaultValue` | `"user"` |
| Relationships | `stmt.Schema.Relationships` | has_many, belongs_to, etc. |

### Supported GORM Tags

The introspector recognizes standard GORM struct tags:

```go
type User struct {
    ID        uint      `gorm:"primarykey"`
    Name      string    `gorm:"size:100;not null"`
    Email     string    `gorm:"size:200;uniqueIndex;not null"`
    Role      string    `gorm:"size:50;default:user"`
    Active    bool      `gorm:"default:true"`
    CreatedAt time.Time
    UpdatedAt time.Time
    Posts     []Post    `gorm:"foreignKey:AuthorID"`
    Profile   *Profile  `gorm:"foreignKey:UserID"`
}
```

Recognized tag components:
- `primarykey` — Marks the primary key column
- `not null` — Marks column as non-nullable
- `size:N` — Column length (informational)
- `default:value` — Default value
- `uniqueIndex` — Unique index
- `index` — Regular index
- `type:text` — Explicit column type
- `foreignKey:FieldName` — Specifies the foreign key field for relationships
- `many2many:table_name` — Join table for many-to-many relationships

### Relationship Detection

GORM Studio detects four relationship types:

| Type | Go Pattern | Example |
|------|-----------|---------|
| `has_one` | `*Model` with foreignKey | `Profile *Profile \`gorm:"foreignKey:UserID"\`` |
| `has_many` | `[]Model` with foreignKey | `Posts []Post \`gorm:"foreignKey:AuthorID"\`` |
| `belongs_to` | Field + FK column | `Author User \`gorm:"foreignKey:AuthorID"\`` |
| `many_to_many` | `[]Model` with many2many tag | `Tags []Tag \`gorm:"many2many:post_tags"\`` |

For each relationship, the introspector records:
- **Name** — The Go field name (e.g., `"Posts"`)
- **Type** — Relationship type (`has_one`, `has_many`, `belongs_to`, `many_to_many`)
- **Table** — The related table name
- **ForeignKey** — The foreign key column name
- **ReferenceKey** — The referenced primary key column name
- **JoinTable** — For many-to-many, the join table name

## Direct Database Introspection

### SQLite

Uses `PRAGMA` statements:

```sql
-- List tables
SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'

-- Column info for each table
PRAGMA table_info('table_name')

-- Foreign key info
PRAGMA foreign_key_list('table_name')
```

Extracts: column names, types, nullability, default values, primary keys, and foreign key relationships.

### PostgreSQL

Uses `information_schema`:

```sql
-- List tables
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public' AND table_type = 'BASE TABLE'

-- Column info
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns WHERE table_name = ?
```

### MySQL

Uses `information_schema`:

```sql
-- List tables
SELECT TABLE_NAME FROM information_schema.tables WHERE table_schema = DATABASE()

-- Column info
SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT
FROM information_schema.columns WHERE table_name = ? AND table_schema = DATABASE()
```

MySQL introspection also detects primary keys (`COLUMN_KEY = 'PRI'`) and potential foreign keys (`COLUMN_KEY = 'MUL'`).

## Merging Strategy

When both sources provide information for the same table, `mergeTableInfo()` applies these rules:

1. **Table name** — From the GORM model
2. **Relationships** — From the GORM model (database introspection doesn't detect GORM-style relationships)
3. **Primary keys** — From the GORM model
4. **Columns** — Iterated from the GORM model, enhanced with DB info:
   - If the model column has no type, the DB type is used
   - If the DB says a column is a foreign key but the model doesn't, the FK info is added
5. **Tables only in DB** — Included as-is (e.g., join tables, legacy tables without Go models)
6. **Tables only in models** — Included as-is (useful before migration)

## Data Types

### Schema Structs

```go
// TableInfo represents a database table
type TableInfo struct {
    Name        string         `json:"name"`
    Columns     []ColumnInfo   `json:"columns"`
    Relations   []RelationInfo `json:"relations"`
    RowCount    int64          `json:"row_count"`
    PrimaryKeys []string       `json:"primary_keys"`
}

// ColumnInfo represents a database column
type ColumnInfo struct {
    Name         string `json:"name"`
    Type         string `json:"type"`           // DB type (e.g., "integer", "text", "varchar")
    GoType       string `json:"go_type"`        // Go type (e.g., "uint", "string", "time.Time")
    IsPrimaryKey bool   `json:"is_primary_key"`
    IsNullable   bool   `json:"is_nullable"`
    IsForeignKey bool   `json:"is_foreign_key"`
    ForeignTable string `json:"foreign_table"`  // Referenced table name
    ForeignKey   string `json:"foreign_key"`    // Referenced column name
    Default      string `json:"default"`
}

// RelationInfo represents a relationship between tables
type RelationInfo struct {
    Name         string `json:"name"`
    Type         string `json:"type"`           // has_one, has_many, belongs_to, many_to_many
    Table        string `json:"table"`
    ForeignKey   string `json:"foreign_key"`
    ReferenceKey string `json:"reference_key"`
    JoinTable    string `json:"join_table"`     // Only for many_to_many
}
```

## Schema Caching

The schema is introspected once when `studio.Mount()` is called and cached in memory. To refresh:

- **API** — `POST /api/schema/refresh`
- **UI** — Click the refresh button (↻) in the sidebar

Refreshing re-runs the full introspection process and updates all row counts.

## Limitations and Known Edge Cases

1. **Composite primary keys** — Currently only the first primary key column is used for CRUD operations. Tables with composite PKs will work for browsing but may have issues with single-row operations.

2. **Custom table names** — If you use `TableName()` method overrides in your GORM models, the introspector should detect them correctly via `gorm.Statement.Parse()`.

3. **Embedded structs** — GORM's embedded struct fields are flattened by the parser, so they appear as regular columns.

4. **Join tables** — Many-to-many join tables (e.g., `post_tags`) appear as standalone tables in the database introspection. They won't have GORM model info unless you define an explicit model for them.

5. **Database-only tables** — Tables that exist in the database but have no corresponding Go model will be discovered with column types from the database dialect, but without Go type information or relationship metadata.

6. **Unsupported dialects** — Only SQLite, PostgreSQL, and MySQL are supported for direct database introspection. Other GORM-supported databases (SQL Server, etc.) will fall back to model-only introspection.

7. **Dynamic schema changes** — If tables are created or modified after `Mount()` is called, use the schema refresh endpoint to update the cached schema.
