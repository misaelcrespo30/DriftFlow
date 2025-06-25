package state

import (
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

//
// ========== Tipos ==========
//

// MigrationState representa el estado de una migración en la base de datos.
type MigrationState struct {
	Version   string
	Applied   bool
	AppliedAt *time.Time
}

// TableInfo representa una tabla y sus columnas.
type TableInfo struct {
	Columns map[string]string // nombre de columna → tipo
}

// SchemaState representa el estado estructural de la base de datos.
type SchemaState struct {
	Tables map[string]TableInfo
}

//
// ========== Lectura de estado desde la base de datos ==========
//

// GetAppliedMigrations consulta la tabla schema_migrations y devuelve las migraciones aplicadas.
func GetAppliedMigrations(db *gorm.DB) ([]MigrationState, error) {
	type result struct {
		Version   string
		AppliedAt *time.Time
	}

	var raw []result
	err := db.Raw(`SELECT version, applied_at FROM schema_migrations`).Scan(&raw).Error
	if err != nil {
		return nil, err
	}

	var states []MigrationState
	for _, r := range raw {
		states = append(states, MigrationState{
			Version:   r.Version,
			Applied:   true,
			AppliedAt: r.AppliedAt,
		})
	}
	return states, nil
}

//
// ========== Lectura de migraciones desde disco ==========
//

// GetLocalMigrationVersions devuelve las versiones de migraciones encontradas en el directorio local.
func GetLocalMigrationVersions(migrationsDir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, file := range files {
		base := filepath.Base(file)
		version := strings.TrimSuffix(base, ".up.sql")
		versions = append(versions, version)
	}
	return versions, nil
}

//
// ========== Comparación de estado (drift detection) ==========
//

// DiffMigrationState compara las migraciones aplicadas con las locales.
// Devuelve:
// - missing: migraciones locales que no han sido aplicadas.
// - extra: migraciones aplicadas que no existen en disco.
func DiffMigrationState(local []string, applied []MigrationState) (missing []string, extra []string) {
	appliedMap := map[string]bool{}
	for _, a := range applied {
		appliedMap[a.Version] = true
	}

	localMap := map[string]bool{}
	for _, l := range local {
		localMap[l] = true
		if !appliedMap[l] {
			missing = append(missing, l)
		}
	}

	for _, a := range applied {
		if !localMap[a.Version] {
			extra = append(extra, a.Version)
		}
	}

	return
}

//
// ========== Utilidades para validación o auditoría ==========
//

// HasDrift verifica si existe cualquier diferencia entre local y aplicado.
func HasDrift(local []string, applied []MigrationState) bool {
	missing, extra := DiffMigrationState(local, applied)
	return len(missing) > 0 || len(extra) > 0
}

// IsMigrationApplied verifica si una versión específica ha sido aplicada.
func IsMigrationApplied(version string, applied []MigrationState) bool {
	for _, m := range applied {
		if m.Version == version && m.Applied {
			return true
		}
	}
	return false
}
