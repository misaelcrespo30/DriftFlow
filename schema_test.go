package driftflow

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

type schemaUser struct {
	ID    uint
	Email string
}

func TestExtractSchema(t *testing.T) {
	db := testDB(t)
	if err := db.AutoMigrate(&schemaUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	meta, err := ExtractSchema(db)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	cols, ok := meta["schema_users"]
	if !ok {
		t.Fatalf("missing users table")
	}
	if _, ok := cols["id"]; !ok {
		t.Fatalf("missing id column")
	}
	if _, ok := cols["email"]; !ok {
		t.Fatalf("missing email column")
	}
}
