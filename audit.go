package driftflow

import (
	"time"
	"gorm.io/gorm"
)

// AuditLog represents a single audit entry in the audit_logs table.
type AuditLog struct {
	ID        uint
	Table     string
	Action    string
	Data      string
	CreatedAt time.Time
}

// TableName overrides the default table name
func (AuditLog) TableName() string {
	return "audit_logs"
}

// ListAuditLog returns all audit log entries ordered by ID ascending.
func ListAuditLog(db *gorm.DB) ([]AuditLog, error) {
	var logs []AuditLog
	if err := db.Order("id").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

