package studio

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// ColumnInfo represents a database column
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	GoType       string `json:"go_type,omitempty"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	IsNullable   bool   `json:"is_nullable"`
	IsForeignKey bool   `json:"is_foreign_key"`
	ForeignTable string `json:"foreign_table,omitempty"`
	ForeignKey   string `json:"foreign_key,omitempty"`
	Default      string `json:"default,omitempty"`
}

// RelationInfo represents a relationship between tables
type RelationInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // has_one, has_many, belongs_to, many_to_many
	Table        string `json:"table"`
	ForeignKey   string `json:"foreign_key"`
	ReferenceKey string `json:"reference_key"`
	JoinTable    string `json:"join_table,omitempty"`
}

// TableInfo represents a database table
type TableInfo struct {
	Name        string         `json:"name"`
	Columns     []ColumnInfo   `json:"columns"`
	Relations   []RelationInfo `json:"relations"`
	RowCount    int64          `json:"row_count"`
	PrimaryKeys []string       `json:"primary_keys"`
}

// SchemaInfo holds the complete database schema
type SchemaInfo struct {
	Tables   []TableInfo `json:"tables"`
	Database string      `json:"database"`
	Driver   string      `json:"driver"`
}

// IntrospectSchema discovers the schema using both GORM models and DB introspection
func IntrospectSchema(db *gorm.DB, models []interface{}) (*SchemaInfo, error) {
	schema := &SchemaInfo{
		Tables: make([]TableInfo, 0),
		Driver: db.Dialector.Name(),
	}

	// First, parse GORM models via reflection
	modelTables := make(map[string]*TableInfo)
	for _, model := range models {
		table, err := parseGORMModel(db, model)
		if err != nil {
			continue
		}
		modelTables[table.Name] = table
	}

	// Then, introspect database tables
	dbTables, err := introspectDatabase(db)
	if err != nil {
		// Fall back to model-only info
		for _, table := range modelTables {
			schema.Tables = append(schema.Tables, *table)
		}
		return schema, nil
	}

	// Merge information from both sources
	seen := make(map[string]bool)
	for _, dbTable := range dbTables {
		if modelTable, ok := modelTables[dbTable.Name]; ok {
			// Merge: prefer GORM model info for types, add DB info
			merged := mergeTableInfo(modelTable, &dbTable)
			schema.Tables = append(schema.Tables, *merged)
		} else {
			schema.Tables = append(schema.Tables, dbTable)
		}
		seen[dbTable.Name] = true
	}

	// Add any model tables not found in DB
	for name, table := range modelTables {
		if !seen[name] {
			schema.Tables = append(schema.Tables, *table)
		}
	}

	// Get row counts
	for i := range schema.Tables {
		var count int64
		db.Table(schema.Tables[i].Name).Count(&count)
		schema.Tables[i].RowCount = count
	}

	return schema, nil
}

func parseGORMModel(db *gorm.DB, model interface{}) (*TableInfo, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return nil, err
	}

	table := &TableInfo{
		Name:        stmt.Schema.Table,
		Columns:     make([]ColumnInfo, 0),
		Relations:   make([]RelationInfo, 0),
		PrimaryKeys: make([]string, 0),
	}

	// Parse fields
	for _, field := range stmt.Schema.Fields {
		col := ColumnInfo{
			Name:         field.DBName,
			Type:         string(field.DataType),
			GoType:       field.FieldType.String(),
			IsPrimaryKey: field.PrimaryKey,
			IsNullable:   !field.NotNull,
		}

		if field.PrimaryKey {
			table.PrimaryKeys = append(table.PrimaryKeys, field.DBName)
		}

		if field.HasDefaultValue {
			col.Default = fmt.Sprintf("%v", field.DefaultValue)
		}

		table.Columns = append(table.Columns, col)
	}

	// Parse relationships
	for _, rel := range stmt.Schema.Relationships.Relations {
		ri := RelationInfo{
			Name:  rel.Name,
			Table: rel.FieldSchema.Table,
		}

		switch rel.Type {
		case "has_one":
			ri.Type = "has_one"
		case "has_many":
			ri.Type = "has_many"
		case "belongs_to":
			ri.Type = "belongs_to"
		case "many_to_many":
			ri.Type = "many_to_many"
			if rel.JoinTable != nil {
				ri.JoinTable = rel.JoinTable.Table
			}
		default:
			ri.Type = string(rel.Type)
		}

		if len(rel.References) > 0 {
			ri.ForeignKey = rel.References[0].ForeignKey.DBName
			ri.ReferenceKey = rel.References[0].PrimaryKey.DBName
		}

		// Mark foreign key columns
		for i, col := range table.Columns {
			if col.Name == ri.ForeignKey {
				table.Columns[i].IsForeignKey = true
				table.Columns[i].ForeignTable = ri.Table
				table.Columns[i].ForeignKey = ri.ReferenceKey
			}
		}

		table.Relations = append(table.Relations, ri)
	}

	return table, nil
}

func introspectDatabase(db *gorm.DB) ([]TableInfo, error) {
	dialect := db.Dialector.Name()
	var tables []TableInfo

	switch dialect {
	case "sqlite":
		tables = introspectSQLite(db)
	case "postgres":
		tables = introspectPostgres(db)
	case "mysql":
		tables = introspectMySQL(db)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}

	return tables, nil
}

func introspectSQLite(db *gorm.DB) []TableInfo {
	var tables []TableInfo
	var tableNames []struct {
		Name string `gorm:"column:name"`
	}

	db.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableNames)

	for _, tn := range tableNames {
		table := TableInfo{
			Name:    tn.Name,
			Columns: make([]ColumnInfo, 0),
		}

		var columns []struct {
			CID       int     `gorm:"column:cid"`
			Name      string  `gorm:"column:name"`
			Type      string  `gorm:"column:type"`
			NotNull   int     `gorm:"column:notnull"`
			Default   *string `gorm:"column:dflt_value"`
			PK        int     `gorm:"column:pk"`
		}

		db.Raw(fmt.Sprintf("PRAGMA table_info('%s')", tn.Name)).Scan(&columns)

		for _, col := range columns {
			ci := ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsPrimaryKey: col.PK > 0,
				IsNullable:   col.NotNull == 0,
			}
			if col.Default != nil {
				ci.Default = *col.Default
			}
			if ci.IsPrimaryKey {
				table.PrimaryKeys = append(table.PrimaryKeys, ci.Name)
			}
			table.Columns = append(table.Columns, ci)
		}

		// Get foreign keys
		var fks []struct {
			ID    int    `gorm:"column:id"`
			Seq   int    `gorm:"column:seq"`
			Table string `gorm:"column:table"`
			From  string `gorm:"column:from"`
			To    string `gorm:"column:to"`
		}
		db.Raw(fmt.Sprintf("PRAGMA foreign_key_list('%s')", tn.Name)).Scan(&fks)

		for _, fk := range fks {
			for i, col := range table.Columns {
				if col.Name == fk.From {
					table.Columns[i].IsForeignKey = true
					table.Columns[i].ForeignTable = fk.Table
					table.Columns[i].ForeignKey = fk.To
				}
			}
		}

		tables = append(tables, table)
	}

	return tables
}

func introspectPostgres(db *gorm.DB) []TableInfo {
	var tables []TableInfo
	var tableNames []struct {
		Name string `gorm:"column:table_name"`
	}

	db.Raw(`SELECT table_name FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`).Scan(&tableNames)

	for _, tn := range tableNames {
		table := TableInfo{
			Name:    tn.Name,
			Columns: make([]ColumnInfo, 0),
		}

		var columns []struct {
			Name     string  `gorm:"column:column_name"`
			Type     string  `gorm:"column:data_type"`
			Nullable string  `gorm:"column:is_nullable"`
			Default  *string `gorm:"column:column_default"`
		}

		db.Raw(`SELECT column_name, data_type, is_nullable, column_default 
			FROM information_schema.columns WHERE table_name = ?`, tn.Name).Scan(&columns)

		for _, col := range columns {
			ci := ColumnInfo{
				Name:       col.Name,
				Type:       col.Type,
				IsNullable: col.Nullable == "YES",
			}
			if col.Default != nil {
				ci.Default = *col.Default
			}
			table.Columns = append(table.Columns, ci)
		}

		tables = append(tables, table)
	}

	return tables
}

func introspectMySQL(db *gorm.DB) []TableInfo {
	var tables []TableInfo
	var tableNames []struct {
		Name string `gorm:"column:TABLE_NAME"`
	}

	db.Raw(`SELECT TABLE_NAME FROM information_schema.tables WHERE table_schema = DATABASE()`).Scan(&tableNames)

	for _, tn := range tableNames {
		table := TableInfo{
			Name:    tn.Name,
			Columns: make([]ColumnInfo, 0),
		}

		var columns []struct {
			Name     string  `gorm:"column:COLUMN_NAME"`
			Type     string  `gorm:"column:COLUMN_TYPE"`
			Nullable string  `gorm:"column:IS_NULLABLE"`
			Key      string  `gorm:"column:COLUMN_KEY"`
			Default  *string `gorm:"column:COLUMN_DEFAULT"`
		}

		db.Raw(`SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT 
			FROM information_schema.columns WHERE table_name = ? AND table_schema = DATABASE()`, tn.Name).Scan(&columns)

		for _, col := range columns {
			ci := ColumnInfo{
				Name:         col.Name,
				Type:         col.Type,
				IsPrimaryKey: col.Key == "PRI",
				IsNullable:   col.Nullable == "YES",
				IsForeignKey: col.Key == "MUL",
			}
			if col.Default != nil {
				ci.Default = *col.Default
			}
			if ci.IsPrimaryKey {
				table.PrimaryKeys = append(table.PrimaryKeys, ci.Name)
			}
			table.Columns = append(table.Columns, ci)
		}

		tables = append(tables, table)
	}

	return tables
}

func mergeTableInfo(modelTable, dbTable *TableInfo) *TableInfo {
	merged := &TableInfo{
		Name:        modelTable.Name,
		Relations:   modelTable.Relations,
		PrimaryKeys: modelTable.PrimaryKeys,
		Columns:     make([]ColumnInfo, 0),
	}

	dbColMap := make(map[string]ColumnInfo)
	for _, col := range dbTable.Columns {
		dbColMap[col.Name] = col
	}

	for _, modelCol := range modelTable.Columns {
		col := modelCol
		if dbCol, ok := dbColMap[col.Name]; ok {
			if col.Type == "" {
				col.Type = dbCol.Type
			}
			if !col.IsForeignKey && dbCol.IsForeignKey {
				col.IsForeignKey = true
				col.ForeignTable = dbCol.ForeignTable
				col.ForeignKey = dbCol.ForeignKey
			}
		}
		merged.Columns = append(merged.Columns, col)
	}

	return merged
}

// GetGoType returns the reflect.Type for a model by table name
func GetGoType(models []interface{}, db *gorm.DB, tableName string) reflect.Type {
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			continue
		}
		if stmt.Schema.Table == tableName {
			t := reflect.TypeOf(model)
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			return t
		}
	}
	return nil
}

// FindModelForTable returns the model interface for a given table name
func FindModelForTable(models []interface{}, db *gorm.DB, tableName string) interface{} {
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			continue
		}
		if stmt.Schema.Table == tableName {
			return model
		}
	}
	return nil
}

// IsValidTable checks if a table name exists
func IsValidTable(schema *SchemaInfo, tableName string) bool {
	for _, t := range schema.Tables {
		if strings.EqualFold(t.Name, tableName) {
			return true
		}
	}
	return false
}
