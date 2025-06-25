package driftflow

import (
	"time"

	"gorm.io/gorm"
)

// AuditLog represents a row in the schema_audit_log table.
type AuditLog struct {
	ID        uint `gorm:"primaryKey"`
	Version   string
	Action    string
	User      string
	Host      string
	Timestamp time.Time `gorm:"autoCreateTime"`
}

// TableName sets the table name for AuditLog.
func (AuditLog) TableName() string {
	return "schema_audit_log"
}

// EnsureAuditTable creates the schema_audit_log table if it does not exist.
func EnsureAuditTable(db *gorm.DB) error {
	return db.AutoMigrate(&AuditLog{})
}
