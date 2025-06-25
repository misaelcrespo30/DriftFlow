package driftflow

import (
	"testing"
<<<<<<< HEAD
	"time"
=======
>>>>>>> codex/crear-función-listauditlog

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

<<<<<<< HEAD
func TestListAuditLog(t *testing.T) {
=======
func setupAuditDB(t *testing.T) *gorm.DB {
>>>>>>> codex/crear-función-listauditlog
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
<<<<<<< HEAD
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
=======
	if err := db.Exec(`CREATE TABLE schema_audit_log (
                id integer primary key,
                message text,
                timestamp datetime
        );`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestListAuditLog(t *testing.T) {
	db := setupAuditDB(t)
	// insert records with timestamps
	db.Exec(`INSERT INTO schema_audit_log (message, timestamp) VALUES (?, datetime('now', '-1 second'))`, "first")
	db.Exec(`INSERT INTO schema_audit_log (message, timestamp) VALUES (?, datetime('now'))`, "second")

	records, err := ListAuditLog(db)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if !records[0].Timestamp.Before(records[1].Timestamp) && !records[0].Timestamp.Equal(records[1].Timestamp) {
		t.Fatalf("records not ordered by timestamp")
>>>>>>> codex/crear-función-listauditlog
	}
}
