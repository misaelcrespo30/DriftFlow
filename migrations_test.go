package driftflow

import (
	"os"
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func writeMigration(t *testing.T, dir, name, upSQL, downSQL string) {
	if err := os.WriteFile(filepath.Join(dir, name+".up.sql"), []byte(upSQL), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".down.sql"), []byte(downSQL), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUpAndDown(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_create_users", "CREATE TABLE users(id integer primary key, name text);", "DROP TABLE users;")
	writeMigration(t, dir, "002_create_addresses", "CREATE TABLE addresses(id integer primary key);", "DROP TABLE addresses;")

	db := setupDB(t)

	if err := Up(db, dir); err != nil {
		t.Fatalf("up: %v", err)
	}

	if !db.Migrator().HasTable("users") || !db.Migrator().HasTable("addresses") {
		t.Fatalf("tables not created")
	}

	if err := Down(db, dir, "001_create_users"); err != nil {
		t.Fatalf("down: %v", err)
	}

	if db.Migrator().HasTable("addresses") {
		t.Fatalf("addresses table should be dropped")
	}
	if !db.Migrator().HasTable("users") {
		t.Fatalf("users table should remain")
	}
}
