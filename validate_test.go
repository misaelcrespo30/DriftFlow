package driftflow

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMigration crea el archivo .sql con contenido Up/Down.
func writeMigration(t *testing.T, dir, name, upSQL, downSQL string) {
	t.Helper()
	path := filepath.Join(dir, name+".sql")
	content := formatMigrationFile(upSQL, downSQL)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

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
	if err := os.WriteFile(filepath.Join(dir, "002_missing.sql"), []byte("CREATE TABLE test(id int);"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Validate(dir); err == nil {
		t.Fatalf("expected missing down error")
	}
}

func TestValidateNamingConvention(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "003_BadName", "CREATE TABLE Users(id int);", "DROP TABLE Users;")

	if err := Validate(dir); err == nil {
		t.Fatalf("expected naming convention error")
	}
}
