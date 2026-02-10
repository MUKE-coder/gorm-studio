# Contributing

Thank you for your interest in contributing to GORM Studio! This guide covers the project structure, development workflow, and guidelines.

## Project Structure

```
gorm-studio/
├── go.mod                  # Go module definition
├── main.go                 # Demo application with sample models and seed data
├── README.md               # Project README
├── docs/                   # Documentation
│   ├── getting-started.md
│   ├── configuration.md
│   ├── api-reference.md
│   ├── schema-introspection.md
│   ├── security.md
│   ├── contributing.md
│   └── examples/
│       ├── with-postgres.md
│       ├── with-mysql.md
│       ├── with-auth.md
│       ├── with-existing-app.md
│       └── read-only-viewer.md
└── studio/                 # The library package
    ├── studio.go           # Mount() function and Config struct
    ├── schema.go           # Schema introspection (DB + GORM reflection)
    ├── handlers.go         # REST API handlers (CRUD, filtering, SQL)
    └── frontend.go         # Embedded React SPA
```

### Key Files

| File | Purpose |
|------|---------|
| `studio/studio.go` | Entry point. `Mount()` registers all routes on a Gin engine. Defines `Config` struct. |
| `studio/schema.go` | Schema discovery. Parses GORM models via reflection and queries DB system tables. Merges both sources. |
| `studio/handlers.go` | API handlers. All REST endpoints for CRUD, relations, SQL execution. Helper functions for validation. |
| `studio/frontend.go` | Frontend. `GetFrontendHTML()` returns the complete React SPA as an HTML string. |
| `main.go` | Demo app. Creates a SQLite DB with 5 related models, seeds data, mounts the studio. |

## Development Setup

### Prerequisites

- Go 1.21+
- Git

### Getting Started

```bash
# Clone the repository
git clone https://github.com/MUKE-coder/gorm-studio.git
cd gorm-studio

# Install dependencies
go mod tidy

# Run the demo
go run main.go
```

Open http://localhost:8080/studio to see the UI.

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test
go test -v -run TestGetSchema ./studio/

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...
```

## How to Modify the Frontend

The frontend is currently a single-file React application embedded in `studio/frontend.go` as a Go string returned by `GetFrontendHTML()`.

### Editing the Frontend

1. Open `studio/frontend.go`
2. The HTML, CSS, and React JSX are all in the template string
3. CSS styles are in the `<style>` block
4. React components are in the `<script type="text/babel">` block
5. Make your changes
6. Run `go run main.go` and refresh the browser to see changes

### Frontend Architecture

The React app uses these main components:

| Component | Purpose |
|-----------|---------|
| `App` | Root component. Manages schema state, active table, view mode. |
| `DataTable` | Data grid. Handles pagination, sorting, filtering, search, CRUD operations. |
| `SQLEditor` | SQL editor. Text area with execution, results display, and query history. |
| `RecordForm` | Form component used in create/edit modals. |
| `Modal` | Reusable modal overlay. |
| `Toast` | Toast notification component. |

### Frontend Notes

- The frontend uses Babel standalone for in-browser JSX transpilation (this is a known performance issue — see roadmap)
- React 18 is loaded from CDN
- Fonts: DM Sans (UI) and JetBrains Mono (code/data)
- All CSS is in the same file using CSS custom properties (variables) for theming
- The API base URL is injected via `window.__STUDIO_CONFIG__` from the Go template

## Code Style Guidelines

### Go

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use meaningful variable names
- Add godoc comments to all exported types and functions
- Error handling: always wrap errors with context using `fmt.Errorf("operation: %w", err)`
- Use table-driven tests where applicable

```go
// Good
func (h *Handlers) GetRow(c *gin.Context) {
    tableName := c.Param("table")
    if !IsValidTable(h.Schema, tableName) {
        c.JSON(http.StatusNotFound, gin.H{"error": "table not found"})
        return
    }
    // ...
}

// Good error wrapping
if err != nil {
    return fmt.Errorf("introspecting schema: %w", err)
}
```

### Frontend (React/JSX)

- Functional components with hooks only
- No TypeScript (for now — single-file constraint)
- Component names in PascalCase
- Event handlers prefixed with `handle` (e.g., `handleSort`, `handleDelete`)
- API calls through the `api()` helper function
- Use CSS custom properties for colors/spacing

### Git Commits

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add column resizing to data table
fix: prevent SQL injection in sort column names
docs: add PostgreSQL setup example
refactor: extract validation into middleware
test: add handler unit tests
chore: update dependencies
```

## Pull Request Process

1. **Fork** the repository
2. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
3. **Make your changes** following the code style guidelines
4. **Add tests** for new functionality
5. **Run tests** and ensure they pass:
   ```bash
   go test ./...
   ```
6. **Commit** using conventional commit format
7. **Push** to your fork and open a Pull Request
8. **Describe** your changes in the PR description

### PR Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Code follows Go conventions (`gofmt`, `go vet`)
- [ ] New exports have godoc comments
- [ ] Frontend changes tested in browser
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventional commits

## Reporting Issues

When reporting bugs, please include:
- Go version (`go version`)
- Database driver and version
- Steps to reproduce
- Expected vs actual behavior
- Error messages or logs

## Feature Requests

Feature requests are welcome! Please describe:
- The use case
- How you imagine it working
- Any alternatives you've considered
