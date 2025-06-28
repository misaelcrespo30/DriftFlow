package driftflow

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"

	"github.com/misaelcrespo30/DriftFlow/config"
)

// SchemaMigration represents a row in the schema_migrations table.
type SchemaMigration struct {
	ID        uint      `gorm:"primaryKey"`
	Version   string    `gorm:"uniqueIndex"`
	AppliedAt time.Time `gorm:"autoCreateTime"`
}

// ensureMigrationsTable creates the schema_migrations table if it does not exist.
func ensureMigrationsTable(db *gorm.DB) error {
	return db.AutoMigrate(&SchemaMigration{})
}

// readMigrationFiles returns the up and down migration files sorted by name.
func readMigrationFiles(dir string) (ups, downs []string, err error) {
	ups, err = filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return nil, nil, err
	}
	downs, err = filepath.Glob(filepath.Join(dir, "*.down.sql"))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(ups)
	sort.Strings(downs)
	return ups, downs, nil
}

func migrationVersion(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".up.sql")
	base = strings.TrimSuffix(base, ".down.sql")
	return base
}

// recordMigration inserts a migration record if it doesn't already exist.
func recordMigration(db *gorm.DB, version string) error {
	m := SchemaMigration{Version: version}
	return db.Where("version = ?", version).FirstOrCreate(&m).Error
}

// removeMigration removes a migration record by version.
func removeMigration(db *gorm.DB, version string) error {
	return db.Where("version = ?", version).Delete(&SchemaMigration{}).Error
}

// toSnakeCase converts CamelCase names to snake_case.
func toSnakeCase(s string) string {
	var out []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

// getTagValue extracts a value for key from a gorm struct tag.
func getTagValue(tag, key string) string {
	parts := strings.Split(tag, ";")
	prefix := key + ":"
	for _, p := range parts {
		if strings.HasPrefix(p, prefix) {
			return strings.TrimPrefix(p, prefix)
		}
	}
	return ""
}

// sqlTypeOf provides a simple mapping from Go types to SQL types.
func sqlTypeOf(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return "timestamp"
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t.Kind() == reflect.Int64 {
			return "bigint"
		}
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if t.Kind() == reflect.Uint64 {
			return "bigint"
		}
		return "integer"
	case reflect.Float32:
		return "real"
	case reflect.Float64:
		return "double precision"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "text"
	default:
		return "text"
	}
}

// Up applies all pending migrations found in dir.
func Up(db *gorm.DB, dir string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	ups, _, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	for _, f := range ups {
		version := migrationVersion(f)
		var count int64
		if err := db.Model(&SchemaMigration{}).Where("version = ?", version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("apply %s: %w", f, err)
		}
		if err := recordMigration(db, version); err != nil {
			return err
		}
		LogAuditEvent(db, version, "apply")
	}
	return nil
}

// Down rolls back migrations until targetVersion is reached.
func Down(db *gorm.DB, dir string, targetVersion string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_, downs, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	downMap := make(map[string]string)
	for _, f := range downs {
		downMap[migrationVersion(f)] = f
	}
	var applied []SchemaMigration
	if err := db.Order("id desc").Find(&applied).Error; err != nil {
		return err
	}
	for _, m := range applied {
		if m.Version == targetVersion {
			break
		}
		file, ok := downMap[m.Version]
		if !ok {
			return fmt.Errorf("missing down file for %s", m.Version)
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("revert %s: %w", file, err)
		}
		if err := removeMigration(db, m.Version); err != nil {
			return err
		}
		LogAuditEvent(db, m.Version, "rollback")
	}
	return nil
}

// DownSteps rolls back the most recent N migrations. If steps is less than 1
// or greater than the number of applied migrations, all applied migrations are
// rolled back.
func DownSteps(db *gorm.DB, dir string, steps int) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_, downs, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	downMap := make(map[string]string)
	for _, f := range downs {
		downMap[migrationVersion(f)] = f
	}
	var applied []SchemaMigration
	if err := db.Order("id desc").Find(&applied).Error; err != nil {
		return err
	}
	if steps < 1 || steps > len(applied) {
		steps = len(applied)
	}
	for i := 0; i < steps; i++ {
		m := applied[i]
		file, ok := downMap[m.Version]
		if !ok {
			return fmt.Errorf("missing down file for %s", m.Version)
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("revert %s: %w", file, err)
		}
		if err := removeMigration(db, m.Version); err != nil {
			return err
		}
		LogAuditEvent(db, m.Version, "rollback")
	}
	return nil
}

// GenerateMigrations is a placeholder for automatic generation.
// GenerateMigrations inspects the database schema and writes migration files
// for any new tables or columns found in the provided models. Only basic
// additions are handled.
func GenerateMigrations(db *gorm.DB, models []interface{}, dir string) error {
	fmt.Printf("GenerateMigrations dir=%s\n", dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for i, m := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return err
		}

		table := stmt.Schema.Table
		var cols []string
		for _, f := range stmt.Schema.Fields {
			if f.Hidden {
				continue
			}
			name := f.DBName
			if name == "" {
				name = toSnakeCase(f.Name)
			}
			cols = append(cols, fmt.Sprintf("%s %s", name, sqlTypeOf(f.FieldType)))
		}
		if len(cols) == 0 {
			continue
		}

		upSQL := fmt.Sprintf("CREATE TABLE %s (\n  %s\n);\n", table, strings.Join(cols, ",\n  "))
		downSQL := fmt.Sprintf("DROP TABLE %s;\n", table)

		prefix := fmt.Sprintf("%04d_%s", i+1, table)
		upPath := filepath.Join(dir, prefix+".up.sql")
		downPath := filepath.Join(dir, prefix+".down.sql")
		if _, err := os.Stat(upPath); os.IsNotExist(err) {
			if err := os.WriteFile(upPath, []byte(upSQL), 0o644); err != nil {
				return err
			}
		}
		if _, err := os.Stat(downPath); os.IsNotExist(err) {
			if err := os.WriteFile(downPath, []byte(downSQL), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// Migrate generates migrations from the given models and then applies all
// pending migration files.
func Migrate(db *gorm.DB, dir string, models []interface{}) error {
	if err := GenerateMigrations(db, models, dir); err != nil {
		return err
	}
	return Up(db, dir)
}
