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

// modelsSchema builds a schemaInfo map from the provided models.
func modelsSchema(db *gorm.DB, models []interface{}) (schemaInfo, error) {
	s := make(schemaInfo)
	for _, m := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return nil, err
		}
		cols := make(tableInfo)
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" || f.IgnoreMigration {
				continue
			}
			name := f.DBName
			if name == "" {
				name = toSnakeCase(f.Name)
			}
			cols[name] = sqlTypeOf(f.FieldType)
		}
		if len(cols) > 0 {
			s[stmt.Schema.Table] = cols
		}
	}
	return s, nil
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

	dbSchema, err := schemaMap(db)
	if err != nil {
		return err
	}

	modelSchema, err := modelsSchema(db, models)
	if err != nil {
		return err
	}

	diffs := diffSchemas(dbSchema, modelSchema)
	if len(diffs) == 0 {
		return nil
	}

	var upStmts, downStmts []string
	for _, d := range diffs {
		switch {
		case strings.HasPrefix(d, "[+] table "):
			tbl := strings.TrimPrefix(d, "[+] table ")
			cols := modelSchema[tbl]
			var defs []string
			for c, t := range cols {
				defs = append(defs, fmt.Sprintf("%s %s", c, t))
			}
			sort.Strings(defs)
			upStmts = append(upStmts, fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", tbl, strings.Join(defs, ",\n  ")))
			downStmts = append(downStmts, fmt.Sprintf("DROP TABLE %s;", tbl))
		case strings.HasPrefix(d, "[-] table "):
			tbl := strings.TrimPrefix(d, "[-] table ")
			cols := dbSchema[tbl]
			var defs []string
			for c, t := range cols {
				defs = append(defs, fmt.Sprintf("%s %s", c, t))
			}
			sort.Strings(defs)
			upStmts = append(upStmts, fmt.Sprintf("DROP TABLE %s;", tbl))
			downStmts = append(downStmts, fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", tbl, strings.Join(defs, ",\n  ")))
		case strings.HasPrefix(d, "[+] column "):
			rest := strings.TrimPrefix(d, "[+] column ")
			parts := strings.Split(rest, ".")
			tbl, col := parts[0], parts[1]
			typ := modelSchema[tbl][col]
			upStmts = append(upStmts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tbl, col, typ))
			downStmts = append(downStmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tbl, col))
		case strings.HasPrefix(d, "[-] column "):
			rest := strings.TrimPrefix(d, "[-] column ")
			parts := strings.Split(rest, ".")
			tbl, col := parts[0], parts[1]
			typ := dbSchema[tbl][col]
			upStmts = append(upStmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tbl, col))
			downStmts = append(downStmts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tbl, col, typ))
		case strings.HasPrefix(d, "[~] column "):
			rest := strings.TrimPrefix(d, "[~] column ")
			parts := strings.Split(rest, " ")
			tblCol := parts[0]
			fromType := parts[1]
			toType := parts[3]
			tp := strings.Split(tblCol, ".")
			tbl, col := tp[0], tp[1]
			upStmts = append(upStmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", tbl, col, toType))
			downStmts = append(downStmts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", tbl, col, fromType))
		}
	}

	timestamp := time.Now().UTC().Format("20060102150405")
	prefix := fmt.Sprintf("%s_auto", timestamp)
	upPath := filepath.Join(dir, prefix+".up.sql")
	downPath := filepath.Join(dir, prefix+".down.sql")

	if err := os.WriteFile(upPath, []byte(strings.Join(upStmts, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(downPath, []byte(strings.Join(downStmts, "\n")+"\n"), 0o644); err != nil {
		return err
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
