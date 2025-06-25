package driftflow

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SchemaAuditLog represents a row in the schema_audit_log table.
type SchemaAuditLog struct {
	ID       uint `gorm:"primaryKey"`
	Version  string
	Action   string
	User     string
	Host     string
	LoggedAt time.Time `gorm:"autoCreateTime"`
}

// TableName specifies the database table name for SchemaAuditLog.
func (SchemaAuditLog) TableName() string { return "schema_audit_log" }

// LogAuditEvent inserts an audit log entry for the given migration version and action.
// It detects the current user and hostname automatically.
func LogAuditEvent(db *gorm.DB, version string, action string) {
	user := os.Getenv("USER")
	if user == "" {
		if out, err := exec.Command("whoami").Output(); err == nil {
			user = strings.TrimSpace(string(out))
		}
	}

	host, _ := os.Hostname()

	entry := SchemaAuditLog{
		Version: version,
		Action:  action,
		User:    user,
		Host:    host,
	}

	// Ignore error as function signature does not return it.
	_ = db.Create(&entry).Error
}
