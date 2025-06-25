package driftflow

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSchema(t *testing.T, db *gorm.DB, stmts []string) {
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("exec: %v", err)
		}
	}
}

func TestCompareDBs(t *testing.T) {
	db1, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open1: %v", err)
	}
	db2, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open2: %v", err)
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
		t.Fatalf("compare: %v", err)
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
		t.Fatalf("missing diff for old table")
	}
	if !has("[+] table posts") {
		t.Fatalf("missing diff for posts table")
	}
	if !has("[+] column users.email") {
		t.Fatalf("missing diff for users.email")
	}
}
