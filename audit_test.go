package driftflow

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLogAuditEvent(t *testing.T) {
	// Preparar base de datos SQLite en memoria
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	// Crear tabla de auditoría
	if err := db.AutoMigrate(&SchemaAuditLog{}); err != nil {
		t.Fatalf("failed to migrate schema_audit_log: %v", err)
	}

	// Simular usuario
	t.Setenv("USER", "tester")

	// Ejecutar LogAuditEvent
	LogAuditEvent(db, "20240625_init", "apply")

	// Leer el log insertado
	var log SchemaAuditLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	// Verificaciones
	if log.Version != "20240625_init" {
		t.Errorf("expected version '20240625_init', got %s", log.Version)
	}
	if log.Action != "apply" {
		t.Errorf("expected action 'apply', got %s", log.Action)
	}
	if log.User != "tester" {
		t.Errorf("expected user 'tester', got %s", log.User)
	}
	if log.Host == "" {
		t.Errorf("expected host to be set, got empty")
	}
	if log.LoggedAt.IsZero() {
		t.Errorf("expected LoggedAt timestamp to be set")
	}
}
