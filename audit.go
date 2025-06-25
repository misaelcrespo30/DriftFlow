package driftflow

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SchemaAuditLog representa una fila en la tabla de auditoría
type SchemaAuditLog struct {
	ID       uint      `gorm:"primaryKey"`
	Version  string    // Versión de la migración o nombre del seed
	Action   string    // apply, rollback, seed, etc.
	User     string    // usuario que ejecuta
	Host     string    // hostname del sistema
	LoggedAt time.Time `gorm:"autoCreateTime"` // timestamp automático
}

// TableName especifica la tabla usada por GORM
func (SchemaAuditLog) TableName() string {
	return "schema_audit_log"
}

// LogAuditEvent inserta una nueva entrada de auditoría en la tabla.
// Detecta automáticamente el usuario y hostname del sistema.
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

	_ = db.Create(&entry).Error // ignoramos el error por simplicidad
}

// ListAuditLog devuelve todas las entradas de auditoría ordenadas por timestamp
func ListAuditLog(db *gorm.DB) ([]SchemaAuditLog, error) {
	var logs []SchemaAuditLog
	if err := db.Order("logged_at").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
