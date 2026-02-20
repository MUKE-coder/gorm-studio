package studio

import (
	"testing"
)

func TestParseCreateStatements_SQLite(t *testing.T) {
	sql := `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(100) NOT NULL,
			email TEXT,
			active BOOLEAN DEFAULT true
		);
	`
	tables, err := ParseCreateStatements(sql, "sqlite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	table := tables[0]
	if table.Name != "users" {
		t.Errorf("expected table name 'users', got %q", table.Name)
	}
	if len(table.Columns) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(table.Columns))
	}
	// Check primary key
	if !table.Columns[0].IsPrimaryKey {
		t.Error("expected id to be primary key")
	}
	// Check NOT NULL
	if table.Columns[1].IsNullable {
		t.Error("expected name to be NOT NULL")
	}
	// Check nullable
	if !table.Columns[2].IsNullable {
		t.Error("expected email to be nullable")
	}
}

func TestParseCreateStatements_Postgres(t *testing.T) {
	sql := `
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title VARCHAR(300) NOT NULL,
			body TEXT,
			author_id INTEGER NOT NULL REFERENCES users(id)
		);
	`
	tables, err := ParseCreateStatements(sql, "postgres")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	table := tables[0]
	if table.Name != "posts" {
		t.Errorf("expected table name 'posts', got %q", table.Name)
	}
	// SERIAL should be marked as PK
	if !table.Columns[0].IsPrimaryKey {
		t.Error("expected id (SERIAL) to be primary key")
	}
	// Check inline foreign key
	authorCol := table.Columns[3]
	if !authorCol.IsForeignKey {
		t.Error("expected author_id to have foreign key")
	}
	if authorCol.ForeignTable != "users" {
		t.Errorf("expected foreign table 'users', got %q", authorCol.ForeignTable)
	}
}

func TestParseCreateStatements_MySQL(t *testing.T) {
	sql := "CREATE TABLE `tags` (\n  `id` INT AUTO_INCREMENT,\n  `name` VARCHAR(50) NOT NULL,\n  PRIMARY KEY (`id`)\n);"
	tables, err := ParseCreateStatements(sql, "mysql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if tables[0].Name != "tags" {
		t.Errorf("expected table name 'tags', got %q", tables[0].Name)
	}
	// PRIMARY KEY declared as constraint
	if !tables[0].Columns[0].IsPrimaryKey {
		t.Error("expected id to be primary key from constraint")
	}
}

func TestParseMultipleTables(t *testing.T) {
	sql := `
		CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT, author_id INTEGER);
	`
	tables, err := ParseCreateStatements(sql, "sqlite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tables))
	}
}

func TestParseForeignKeyConstraint(t *testing.T) {
	sql := `
		CREATE TABLE posts (
			id INTEGER PRIMARY KEY,
			author_id INTEGER NOT NULL,
			FOREIGN KEY (author_id) REFERENCES users(id)
		);
	`
	tables, err := ParseCreateStatements(sql, "sqlite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	authorCol := tables[0].Columns[1]
	if !authorCol.IsForeignKey {
		t.Error("expected author_id to be foreign key")
	}
	if authorCol.ForeignTable != "users" {
		t.Errorf("expected foreign table 'users', got %q", authorCol.ForeignTable)
	}
	if authorCol.ForeignKey != "id" {
		t.Errorf("expected foreign key 'id', got %q", authorCol.ForeignKey)
	}
}

func TestDetectDialect(t *testing.T) {
	tests := map[string]string{
		"CREATE TABLE t (id INTEGER PRIMARY KEY AUTOINCREMENT)": "sqlite",
		"CREATE TABLE t (id SERIAL PRIMARY KEY)":                "postgres",
		"CREATE TABLE t (id INT AUTO_INCREMENT)":                "mysql",
		"CREATE TABLE t (id INT)":                               "sqlite",
	}
	for input, expected := range tests {
		result := detectDialect(input)
		if result != expected {
			t.Errorf("detectDialect(%q) = %q, want %q", input[:30], result, expected)
		}
	}
}

func TestRemoveComments(t *testing.T) {
	input := "SELECT 1; -- comment\n/* block */SELECT 2;"
	result := removeComments(input)
	if result == input {
		t.Error("expected comments to be removed")
	}
}
