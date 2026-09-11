package studio

import "testing"

// The fuzz targets below assert only that the hand-rolled parsers never panic,
// hang, or otherwise misbehave on adversarial input. Running `go test` executes
// their seed corpus; `go test -fuzz=Fuzz...` explores further.

func FuzzParseCreateStatements(f *testing.F) {
	f.Add("CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT NOT NULL);")
	f.Add("CREATE TABLE `x` (`a` VARCHAR(255) DEFAULT 'z', b DECIMAL(10,2));")
	f.Add("create table if not exists q (id serial primary key);")
	f.Add("")
	f.Add("CREATE TABLE (")
	f.Add("CREATE TABLE t (a int REFERENCES other(id));")
	f.Fuzz(func(t *testing.T, s string) {
		for _, dialect := range []string{"", "sqlite", "postgres", "mysql"} {
			_, _ = ParseCreateStatements(s, dialect)
		}
	})
}

func FuzzParseDBML(f *testing.F) {
	f.Add("Table users {\n id integer [pk]\n name varchar\n}\n")
	f.Add("Table a {\n x int [ref: > b.id]\n}\nRef: a.x > b.id\n")
	f.Add("")
	f.Add("Table {")
	f.Add("}}}}}")
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parseDBML(s)
	})
}

func FuzzParseGoStructs(f *testing.F) {
	f.Add("type User struct {\n ID uint `gorm:\"primaryKey\"`\n Name string\n}")
	f.Add("type A struct { B []C; D *E }")
	f.Add("")
	f.Add("type struct {")
	f.Add("type X struct {{{{")
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parseGoStructs(s)
	})
}

func FuzzSplitStatements(f *testing.F) {
	f.Add("SELECT 1; SELECT 2;")
	f.Add("INSERT INTO t VALUES ('(', ';'); DELETE FROM t;")
	f.Add("'unterminated")
	f.Add("((((")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		stmts := splitStatements(s)
		// Splitting then removing comments must never panic on the pieces.
		for _, st := range stmts {
			_ = removeComments(st)
		}
	})
}

func FuzzRemoveComments(f *testing.F) {
	f.Add("SELECT 1 -- comment\n/* block */ FROM t")
	f.Add("'string with -- not a comment'")
	f.Add("/* unterminated")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		_ = removeComments(s)
	})
}
