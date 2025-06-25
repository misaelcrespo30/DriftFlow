package driftflow

import (
	"testing"
)

func TestLogAuditEvent(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&SchemaAuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	t.Setenv("USER", "tester")

	LogAuditEvent(db, "001_test", "up")

	var log SchemaAuditLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("read log: %v", err)
	}

	if log.Version != "001_test" || log.Action != "up" {
		t.Fatalf("unexpected log values: %+v", log)
	}
	if log.User != "tester" {
		t.Fatalf("expected user tester, got %s", log.User)
	}
	if log.Host == "" {
		t.Fatalf("expected hostname set")
	}
}
