package driftflow

import (
	"reflect"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//
// ========== Pruebas Unitarias: CompareSchemas (mock con mapas) ==========
//

func TestCompareSchemas_TableMissing(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int"},
	}
	tgt := map[string]map[string]string{}

	expect := []string{"missing table: users"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}

func TestCompareSchemas_ColumnMissing(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int", "name": "text"},
	}
	tgt := map[string]map[string]string{
		"users": {"id": "int"},
	}

	expect := []string{"missing column: users.name"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}

func TestCompareSchemas_ColumnExtra(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int"},
	}
	tgt := map[string]map[string]string{
		"users": {"id": "int", "name": "text"},
	}

	expect := []string{"extra column: users.name"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}

func TestCompareSchemas_TypeMismatch(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int"},
	}
	tgt := map[string]map[string]string{
		"users": {"id": "bigint"},
	}

	expect := []string{"type mismatch for users.id: int vs bigint"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}

//
// ========== Prueba de integración real: CompareDBs con SQLite ==========
//

func setupSchema(t *testing.T, db *gorm.DB, stmts []string) {
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("failed to execute schema: %v", err)
		}
	}
}

func TestCompareDBs(t *testing.T) {
	db1, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db1: %v", err)
	}
	db2, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db2: %v", err)
	}

	setupSchema(t, db1, []string{
		"CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT);",
		"CREATE TABLE old(id INTEGER);",
	})
	setupSchema(t, db2, []string{
		"CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT, email TEXT);",
		"CREATE TABLE posts(id INTEGER);",
	})

	diffs, err := CompareDBs(db1, db2)
	if err != nil {
		t.Fatalf("CompareDBs failed: %v", err)
	}

	has := func(want string) bool {
		for _, d := range diffs {
			if d == want {
				return true
			}
		}
		return false
	}

	if !has("[-] table old") {
		t.Errorf("expected diff for missing table 'old'")
	}
	if !has("[+] table posts") {
		t.Errorf("expected diff for added table 'posts'")
	}
	if !has("[+] column users.email") {
		t.Errorf("expected diff for added column 'users.email'")
	}
}
