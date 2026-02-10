package studio

import "fmt"

// ErrTableNotFound is returned when a table name is not found in the schema.
type ErrTableNotFound struct {
	Table string
}

func (e *ErrTableNotFound) Error() string {
	return fmt.Sprintf("table not found: %s", e.Table)
}

// ErrInvalidColumn is returned when a column name is not valid for the given table.
type ErrInvalidColumn struct {
	Table  string
	Column string
}

func (e *ErrInvalidColumn) Error() string {
	return fmt.Sprintf("invalid column %q for table %q", e.Column, e.Table)
}

// ErrRowNotFound is returned when a row with the given primary key is not found.
type ErrRowNotFound struct {
	Table string
	ID    string
}

func (e *ErrRowNotFound) Error() string {
	return fmt.Sprintf("row not found in %s with id %s", e.Table, e.ID)
}

// ErrNoPrimaryKey is returned when no primary key is defined for a table.
type ErrNoPrimaryKey struct {
	Table string
}

func (e *ErrNoPrimaryKey) Error() string {
	return fmt.Sprintf("no primary key found for table %s", e.Table)
}

// ErrRelationNotFound is returned when a relationship name is not found on a table.
type ErrRelationNotFound struct {
	Table    string
	Relation string
}

func (e *ErrRelationNotFound) Error() string {
	return fmt.Sprintf("relation %q not found on table %q", e.Relation, e.Table)
}

// ErrReadOnly is returned when a write operation is attempted in read-only mode.
type ErrReadOnly struct{}

func (e *ErrReadOnly) Error() string {
	return "write operations are disabled in read-only mode"
}

// ErrSQLDisabled is returned when the SQL endpoint is called but SQL is disabled.
type ErrSQLDisabled struct{}

func (e *ErrSQLDisabled) Error() string {
	return "SQL editor is disabled"
}
