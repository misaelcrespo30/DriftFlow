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
	Version  string    // Versión de la migración o seed ejecutado
	Commit   string    // Hash del commit de Git
	Action   string    // Tipo de acción: apply, rollback, seed, etc.
	User     string    // Usuario del sistema
	Host     string    // Nombre del host
	LoggedAt time.Time `gorm:"autoCreateTime"` // Timestamp generado automáticamente
}

// TableName define el nombre de la tabla usada por GORM
func (SchemaAuditLog) TableName() string {
	return "schema_audit_log"
}

// EnsureAuditTable asegura que la tabla schema_audit_log exista.
// Usa AutoMigrate para crearla si no está.
func EnsureAuditTable(db *gorm.DB) error {
	return db.AutoMigrate(&SchemaAuditLog{})
}

func gitCommitHash() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LogAuditEvent inserta una nueva entrada de auditoría.
// Detecta automáticamente el usuario y el hostname del sistema.
func LogAuditEvent(db *gorm.DB, version string, action string) {
	user := os.Getenv("USER")
	if user == "" {
		if out, err := exec.Command("whoami").Output(); err == nil {
			user = strings.TrimSpace(string(out))
		}
	}
	host, _ := os.Hostname()
	commit := gitCommitHash()

	entry := SchemaAuditLog{
		Version: version,
		Commit:  commit,
		Action:  action,
		User:    user,
		Host:    host,
	}

	_ = db.Create(&entry).Error // Ignoramos el error por simplicidad
}

// ListAuditLog retorna todas las entradas de auditoría ordenadas por LoggedAt
func ListAuditLog(db *gorm.DB) ([]SchemaAuditLog, error) {
	var logs []SchemaAuditLog
	if err := db.Order("logged_at").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
