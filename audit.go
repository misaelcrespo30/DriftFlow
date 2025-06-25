package driftflow

import (
	"gorm.io/gorm"
	"time"
)

// AuditRecord represents a row in the schema_audit_log table.
type AuditRecord struct {
	ID        uint
	Timestamp time.Time `gorm:"column:timestamp"`
	// Add additional fields as needed
}

func (AuditRecord) TableName() string { return "schema_audit_log" }

// ListAuditLog returns all audit log records ordered by timestamp.
func ListAuditLog(db *gorm.DB) ([]AuditRecord, error) {
	var records []AuditRecord
	if err := db.Order("timestamp").Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
