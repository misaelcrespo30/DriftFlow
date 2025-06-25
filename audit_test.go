package driftflow

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListAuditLog(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// create table
	if err := db.Exec(`CREATE TABLE audit_logs (id integer primary key autoincrement, table text, action text, data text, created_at datetime)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`INSERT INTO audit_logs(table, action, data, created_at) VALUES ('users','create','{}',?)`, time.Now()).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	logs, err := ListAuditLog(db)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Table != "users" {
		t.Fatalf("expected table users, got %s", logs[0].Table)
	}
}
