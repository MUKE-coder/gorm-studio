# API Reference

GORM Studio exposes a REST API under the configured prefix (default: `/studio/api`). All endpoints return JSON responses.

## Base URL

```
http://localhost:8080/studio/api
```

Replace `/studio` with your configured `Prefix`.

## Common Response Formats

### Success Response

```json
{
  "rows": [...],
  "total": 100,
  "page": 1,
  "page_size": 50,
  "pages": 2
}
```

### Error Response

```json
{
  "error": "description of what went wrong"
}
```

---

## Schema Endpoints

### GET /api/schema

Returns the full database schema including all tables, columns, relationships, and row counts.

**Response:**

```json
{
  "tables": [
    {
      "name": "users",
      "columns": [
        {
          "name": "id",
          "type": "integer",
          "go_type": "uint",
          "is_primary_key": true,
          "is_nullable": false,
          "is_foreign_key": false,
          "default": ""
        },
        {
          "name": "name",
          "type": "string",
          "go_type": "string",
          "is_primary_key": false,
          "is_nullable": false,
          "is_foreign_key": false,
          "default": ""
        }
      ],
      "relations": [
        {
          "name": "Posts",
          "type": "has_many",
          "table": "posts",
          "foreign_key": "author_id",
          "reference_key": "id"
        }
      ],
      "row_count": 10,
      "primary_keys": ["id"]
    }
  ],
  "database": "",
  "driver": "sqlite"
}
```

**Example:**

```bash
curl http://localhost:8080/studio/api/schema
```

### POST /api/schema/refresh

Re-introspects the database schema. Use this after running migrations or modifying the database structure.

**Response:** Same format as `GET /api/schema`.

**Example:**

```bash
curl -X POST http://localhost:8080/studio/api/schema/refresh
```

---

## CRUD Endpoints

### GET /api/tables/:table/rows

Returns paginated rows from a table with support for filtering, sorting, and search.

**URL Parameters:**

| Parameter | Description |
|-----------|-------------|
| `:table`  | Table name (e.g., `users`) |

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | `1` | Page number (1-indexed) |
| `page_size` | int | `50` | Rows per page (1–500) |
| `sort_by` | string | — | Column name to sort by |
| `sort_order` | string | `asc` | Sort direction: `asc` or `desc` |
| `search` | string | — | Full-text search across all text columns |
| `filter_<column>` | string | — | Filter by exact column value. Use `%` for LIKE matching. |

**Response:**

```json
{
  "rows": [
    {"id": 1, "name": "Alice", "email": "alice@example.com", "active": true},
    {"id": 2, "name": "Bob", "email": "bob@example.com", "active": false}
  ],
  "total": 10,
  "page": 1,
  "page_size": 50,
  "pages": 1
}
```

**Examples:**

```bash
# Basic listing
curl "http://localhost:8080/studio/api/tables/users/rows"

# With pagination
curl "http://localhost:8080/studio/api/tables/users/rows?page=2&page_size=10"

# With sorting
curl "http://localhost:8080/studio/api/tables/users/rows?sort_by=name&sort_order=asc"

# With search
curl "http://localhost:8080/studio/api/tables/users/rows?search=alice"

# With column filter (exact match)
curl "http://localhost:8080/studio/api/tables/users/rows?filter_role=admin"

# With column filter (LIKE match)
curl "http://localhost:8080/studio/api/tables/users/rows?filter_name=%25alice%25"
```

**Filtering Details:**

- Prefix query parameters with `filter_` followed by the column name
- Exact match: `filter_role=admin` → `WHERE role = 'admin'`
- LIKE match: `filter_name=%alice%` → `WHERE name LIKE '%alice%'` (URL-encode `%` as `%25`)
- Multiple filters are combined with AND
- Only valid column names (verified against the schema) are accepted

**Search Details:**

The `search` parameter performs a case-insensitive LIKE search across all text-type columns (`text`, `varchar`, `char`, `string`, `nvarchar`, `ntext`, `clob`). The conditions are combined with OR:

```sql
WHERE name LIKE '%search%' OR email LIKE '%search%' OR bio LIKE '%search%'
```

### GET /api/tables/:table/rows/:id

Returns a single row by its primary key value.

**URL Parameters:**

| Parameter | Description |
|-----------|-------------|
| `:table`  | Table name |
| `:id`     | Primary key value |

**Response:** A single row object.

```json
{
  "id": 1,
  "name": "Alice Johnson",
  "email": "alice.johnson@example.com",
  "role": "admin",
  "active": true,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Example:**

```bash
curl http://localhost:8080/studio/api/tables/users/rows/1
```

### POST /api/tables/:table/rows

Creates a new row. Not available in read-only mode.

**URL Parameters:**

| Parameter | Description |
|-----------|-------------|
| `:table`  | Table name |

**Request Body:** JSON object with column values. Only valid column names are accepted; unknown fields are silently ignored.

```json
{
  "name": "New User",
  "email": "new@example.com",
  "role": "user",
  "active": true
}
```

**Response:**

```json
{
  "message": "created",
  "data": {
    "name": "New User",
    "email": "new@example.com",
    "role": "user",
    "active": true
  }
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/studio/api/tables/users/rows \
  -H "Content-Type: application/json" \
  -d '{"name": "New User", "email": "new@example.com", "role": "user", "active": true}'
```

### PUT /api/tables/:table/rows/:id

Updates an existing row by primary key. Not available in read-only mode.

**URL Parameters:**

| Parameter | Description |
|-----------|-------------|
| `:table`  | Table name |
| `:id`     | Primary key value |

**Request Body:** JSON object with the fields to update. The primary key field is automatically removed from the update payload.

```json
{
  "name": "Updated Name",
  "role": "editor"
}
```

**Response:**

```json
{
  "message": "updated",
  "rows_affected": 1
}
```

**Example:**

```bash
curl -X PUT http://localhost:8080/studio/api/tables/users/rows/1 \
  -H "Content-Type: application/json" \
  -d '{"name": "Updated Name", "role": "editor"}'
```

### DELETE /api/tables/:table/rows/:id

Deletes a single row by primary key. Not available in read-only mode.

**URL Parameters:**

| Parameter | Description |
|-----------|-------------|
| `:table`  | Table name |
| `:id`     | Primary key value |

**Response:**

```json
{
  "message": "deleted",
  "rows_affected": 1
}
```

**Example:**

```bash
curl -X DELETE http://localhost:8080/studio/api/tables/users/rows/5
```

### POST /api/tables/:table/rows/bulk-delete

Deletes multiple rows by their primary key values. Not available in read-only mode.

**URL Parameters:**

| Parameter | Description |
|-----------|-------------|
| `:table`  | Table name |

**Request Body:**

```json
{
  "ids": [1, 2, 3, 5]
}
```

**Response:**

```json
{
  "message": "deleted",
  "rows_affected": 4
}
```

**Example:**

```bash
curl -X POST http://localhost:8080/studio/api/tables/users/rows/bulk-delete \
  -H "Content-Type: application/json" \
  -d '{"ids": [1, 2, 3]}'
```

---

## Relation Endpoints

### GET /api/tables/:table/rows/:id/relations/:relation

Returns rows from a related table for a specific row.

**URL Parameters:**

| Parameter   | Description |
|-------------|-------------|
| `:table`    | Source table name |
| `:id`       | Source row's primary key value |
| `:relation` | Relationship name (as defined in the GORM model, e.g., `Posts`, `Author`, `Tags`) |

**Response:**

```json
{
  "rows": [
    {"id": 1, "title": "First Post", "author_id": 1, "published": true},
    {"id": 5, "title": "Another Post", "author_id": 1, "published": false}
  ],
  "relation": {
    "name": "Posts",
    "type": "has_many",
    "table": "posts",
    "foreign_key": "author_id",
    "reference_key": "id"
  },
  "total": 2
}
```

**Supported relationship types:**

| Type | Behavior |
|------|----------|
| `has_one` | Queries related table where `foreign_key = source_id` |
| `has_many` | Queries related table where `foreign_key = source_id` |
| `belongs_to` | Looks up the foreign key value on the source row, then queries the related table |
| `many_to_many` | Joins through the join table to find related rows |

**Example:**

```bash
# Get all posts by user 1
curl http://localhost:8080/studio/api/tables/users/rows/1/relations/Posts

# Get the author of post 3
curl http://localhost:8080/studio/api/tables/posts/rows/3/relations/Author

# Get tags for post 1
curl http://localhost:8080/studio/api/tables/posts/rows/1/relations/Tags
```

---

## SQL Endpoint

### POST /api/sql

Executes a raw SQL query. Not available when `DisableSQL` is enabled.

**Request Body:**

```json
{
  "query": "SELECT * FROM users WHERE role = 'admin'"
}
```

**Response (read query):**

```json
{
  "rows": [
    {"id": 1, "name": "Alice", "email": "alice@example.com", "role": "admin"}
  ],
  "total": 1,
  "columns": ["id", "name", "email", "role", "active", "created_at", "updated_at"],
  "rows_affected": 1,
  "type": "read"
}
```

**Response (write query):**

```json
{
  "rows_affected": 3,
  "message": "query executed successfully",
  "type": "write"
}
```

**Query type detection:**

Queries starting with these keywords are treated as read queries: `SELECT`, `EXPLAIN`, `PRAGMA`, `SHOW`, `DESCRIBE`. All other queries are treated as write queries.

**Example:**

```bash
# Read query
curl -X POST http://localhost:8080/studio/api/tables/../sql \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT name, COUNT(*) as post_count FROM users JOIN posts ON users.id = posts.author_id GROUP BY users.id"}'

# Write query
curl -X POST http://localhost:8080/studio/api/sql \
  -H "Content-Type: application/json" \
  -d '{"query": "UPDATE users SET role = '\''editor'\'' WHERE id = 3"}'
```

---

## Config Endpoint

### GET /api/config

Returns the current studio configuration.

**Response:**

```json
{
  "read_only": false,
  "disable_sql": false,
  "prefix": "/studio"
}
```

**Example:**

```bash
curl http://localhost:8080/studio/api/config
```

---

## Error Codes

| HTTP Status | Meaning |
|-------------|---------|
| `200` | Success |
| `201` | Created (new row) |
| `400` | Bad request (invalid JSON, missing required fields, invalid SQL) |
| `404` | Table not found, row not found, or relation not found |
| `500` | Internal server error (database error) |
