package schemaflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDuplicate(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_init", "CREATE TABLE t1(id int);", "DROP TABLE t1;")
	writeMigration(t, dir, "001_init_again", "CREATE TABLE t2(id int);", "DROP TABLE t2;")
	if err := Validate(dir); err == nil {
		t.Fatalf("expected duplicate error")
	}
}

func TestValidateMissingDown(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "002_missing.up.sql"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Validate(dir); err == nil {
		t.Fatalf("expected missing down error")
	}
}
