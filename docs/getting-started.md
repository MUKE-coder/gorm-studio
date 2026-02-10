# Getting Started

## Prerequisites

- **Go 1.21+** — [Install Go](https://go.dev/dl/)
- **A GORM project** — An existing Go application using [GORM v2](https://gorm.io/)
- **Gin** — [Gin web framework](https://github.com/gin-gonic/gin) (framework-agnostic adapters coming soon)

## Installation

Add GORM Studio to your Go project:

```bash
go get github.com/MUKE-coder/gorm-studio/studio
```

Or if you're working locally, copy the `studio/` directory into your project.

## Minimal Setup

Mount GORM Studio in your Gin application with just a few lines:

```go
package main

import (
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
    db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
    db.AutoMigrate(&User{})

    router := gin.Default()

    // Mount GORM Studio — one line!
    studio.Mount(router, db, []interface{}{&User{}})

    router.Run(":8080")
}
```

Open your browser to [http://localhost:8080/studio](http://localhost:8080/studio) and you'll see the GORM Studio UI.

## Running the Demo

The project includes a demo application with sample models and seed data:

```bash
cd gorm-studio
go mod tidy
go run main.go
```

This starts a server at `http://localhost:8080` with GORM Studio mounted at `/studio`. The demo creates a SQLite database with five related models:

| Model     | Description                        |
|-----------|------------------------------------|
| `User`    | Users with name, email, role       |
| `Profile` | One-to-one with User (bio, avatar) |
| `Post`    | Blog posts authored by users       |
| `Comment` | Comments on posts by users         |
| `Tag`     | Tags with many-to-many on posts    |

The demo seeds 10 users, 15 posts with comments and tags, and creates all the relationships including a `post_tags` join table.

## UI Overview

### Sidebar
The left sidebar shows all discovered tables with their row counts. Use the search bar to filter tables by name. Toggle between the **Tables** view and the **SQL** editor using the buttons at the bottom.

### Data Grid
The main area displays a paginated data grid for the selected table:

- **Primary key columns** are highlighted with a gold key icon
- **Foreign key columns** are shown with a blue link icon — click a value to navigate to the referenced table
- **NULL values** are displayed in italic
- **Boolean values** are rendered as colored dots (green for true, red for false)
- **Sorting** — Click any column header to sort ascending/descending
- **Search** — Use the search bar to full-text search across all text columns

### CRUD Operations
When not in read-only mode:

- **Create** — Click "Add Record" to open the creation modal
- **Edit** — Click the edit icon on any row to modify it
- **Delete** — Click the trash icon to delete a single row, or select multiple rows using checkboxes for bulk delete

### SQL Editor
Switch to the SQL tab to execute raw queries:

- Supports both read (SELECT) and write (INSERT, UPDATE, DELETE) queries
- Press **Ctrl+Enter** (or **Cmd+Enter** on macOS) to execute
- Query history is maintained for the current session

## Next Steps

- [Configuration](configuration.md) — Customize prefix, enable read-only mode, disable SQL editor
- [API Reference](api-reference.md) — Full REST API documentation
- [Security](security.md) — Important security considerations for deployment
