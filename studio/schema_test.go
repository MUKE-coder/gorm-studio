package studio

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test models
type TestUser struct {
	ID        uint      `gorm:"primarykey"`
	Name      string    `gorm:"size:100;not null"`
	Email     string    `gorm:"size:200;uniqueIndex"`
	Active    bool      `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Posts     []TestPost `gorm:"foreignKey:AuthorID"`
}

type TestPost struct {
	ID        uint      `gorm:"primarykey"`
	Title     string    `gorm:"size:300;not null"`
	Body      string    `gorm:"type:text"`
	AuthorID  uint      `gorm:"not null;index"`
	Author    TestUser  `gorm:"foreignKey:AuthorID"`
	CreatedAt time.Time
	Tags      []TestTag `gorm:"many2many:test_post_tags"`
}

type TestTag struct {
	ID    uint       `gorm:"primarykey"`
	Name  string     `gorm:"size:50;uniqueIndex;not null"`
	Posts []TestPost `gorm:"many2many:test_post_tags"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	err = db.AutoMigrate(&TestUser{}, &TestPost{}, &TestTag{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func testModels() []interface{} {
	return []interface{}{&TestUser{}, &TestPost{}, &TestTag{}}
}

func TestIntrospectSchema(t *testing.T) {
	db := setupTestDB(t)
	models := testModels()

	schema, err := IntrospectSchema(db, models)
	if err != nil {
		t.Fatalf("IntrospectSchema failed: %v", err)
	}

	if schema.Driver != "sqlite" {
		t.Errorf("expected driver 'sqlite', got %q", schema.Driver)
	}

	// Should have at least the 3 model tables + the join table
	if len(schema.Tables) < 3 {
		t.Errorf("expected at least 3 tables, got %d", len(schema.Tables))
	}

	// Find test_users table
	var usersTable *TableInfo
	for i := range schema.Tables {
		if schema.Tables[i].Name == "test_users" {
			usersTable = &schema.Tables[i]
			break
		}
	}
	if usersTable == nil {
		t.Fatal("test_users table not found in schema")
	}

	// Check columns exist
	colNames := make(map[string]bool)
	for _, col := range usersTable.Columns {
		colNames[col.Name] = true
	}
	for _, expected := range []string{"id", "name", "email", "active"} {
		if !colNames[expected] {
			t.Errorf("expected column %q in test_users, not found", expected)
		}
	}

	// Check primary key
	if len(usersTable.PrimaryKeys) == 0 || usersTable.PrimaryKeys[0] != "id" {
		t.Errorf("expected primary key 'id', got %v", usersTable.PrimaryKeys)
	}

	// Check that user has relations
	if len(usersTable.Relations) == 0 {
		t.Error("expected at least one relation on test_users")
	}
}

func TestIntrospectSchemaRelationships(t *testing.T) {
	db := setupTestDB(t)
	models := testModels()

	schema, err := IntrospectSchema(db, models)
	if err != nil {
		t.Fatalf("IntrospectSchema failed: %v", err)
	}

	// Find test_posts table
	var postsTable *TableInfo
	for i := range schema.Tables {
		if schema.Tables[i].Name == "test_posts" {
			postsTable = &schema.Tables[i]
			break
		}
	}
	if postsTable == nil {
		t.Fatal("test_posts table not found")
	}

	// Check relations
	relMap := make(map[string]RelationInfo)
	for _, rel := range postsTable.Relations {
		relMap[rel.Name] = rel
	}

	if rel, ok := relMap["Author"]; ok {
		if rel.Type != "belongs_to" {
			t.Errorf("expected Author relation type 'belongs_to', got %q", rel.Type)
		}
	} else {
		t.Error("expected 'Author' relation on test_posts")
	}

	if rel, ok := relMap["Tags"]; ok {
		if rel.Type != "many_to_many" {
			t.Errorf("expected Tags relation type 'many_to_many', got %q", rel.Type)
		}
		if rel.JoinTable != "test_post_tags" {
			t.Errorf("expected join table 'test_post_tags', got %q", rel.JoinTable)
		}
	} else {
		t.Error("expected 'Tags' relation on test_posts")
	}
}

func TestIsValidTable(t *testing.T) {
	db := setupTestDB(t)
	models := testModels()

	schema, err := IntrospectSchema(db, models)
	if err != nil {
		t.Fatalf("IntrospectSchema failed: %v", err)
	}

	tests := []struct {
		table string
		want  bool
	}{
		{"test_users", true},
		{"test_posts", true},
		{"test_tags", true},
		{"nonexistent", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			got := IsValidTable(schema, tt.table)
			if got != tt.want {
				t.Errorf("IsValidTable(%q) = %v, want %v", tt.table, got, tt.want)
			}
		})
	}
}

func TestGetGoType(t *testing.T) {
	db := setupTestDB(t)
	models := testModels()

	userType := GetGoType(models, db, "test_users")
	if userType == nil {
		t.Fatal("expected non-nil type for test_users")
	}
	if userType.Name() != "TestUser" {
		t.Errorf("expected type name 'TestUser', got %q", userType.Name())
	}

	nilType := GetGoType(models, db, "nonexistent")
	if nilType != nil {
		t.Errorf("expected nil for nonexistent table, got %v", nilType)
	}
}

func TestFindModelForTable(t *testing.T) {
	db := setupTestDB(t)
	models := testModels()

	model := FindModelForTable(models, db, "test_users")
	if model == nil {
		t.Fatal("expected non-nil model for test_users")
	}

	model = FindModelForTable(models, db, "nonexistent")
	if model != nil {
		t.Errorf("expected nil for nonexistent table, got %v", model)
	}
}

func TestRowCounts(t *testing.T) {
	db := setupTestDB(t)
	models := testModels()

	// Insert some data
	db.Create(&TestUser{Name: "Alice", Email: "alice@test.com"})
	db.Create(&TestUser{Name: "Bob", Email: "bob@test.com"})

	schema, err := IntrospectSchema(db, models)
	if err != nil {
		t.Fatalf("IntrospectSchema failed: %v", err)
	}

	for _, table := range schema.Tables {
		if table.Name == "test_users" {
			if table.RowCount != 2 {
				t.Errorf("expected 2 rows in test_users, got %d", table.RowCount)
			}
			return
		}
	}
	t.Error("test_users table not found")
}
