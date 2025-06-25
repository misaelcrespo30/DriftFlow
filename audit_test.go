package driftflow

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListAuditLog(t *testing.T) {
	// Crear base de datos SQLite en memoria
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Crear tabla audit_logs
	err = db.Exec(`CREATE TABLE audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		table TEXT,
		action TEXT,
		data TEXT,
		created_at DATETIME
	)`).Error
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Insertar un registro de auditoría
	err = db.Exec(`INSERT INTO audit_logs(table, action, data, created_at) VALUES ('users', 'create', '{}', ?)`, time.Now()).Error
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	// Ejecutar la función que estamos probando
	logs, err := ListAuditLog(db)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// Verificaciones
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Table != "users" {
		t.Fatalf("expected table users, got %s", logs[0].Table)
	}
	if logs[0].Action != "create" {
		t.Fatalf("expected action create, got %s", logs[0].Action)
	}
}
