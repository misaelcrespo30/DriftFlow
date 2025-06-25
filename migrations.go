package driftflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
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

// Up applies all pending migrations found in dir.
func Up(db *gorm.DB, dir string) error {
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

// GenerateMigrations is a placeholder for automatic generation.
// GenerateMigrations inspects the database schema and writes migration files
// for any new tables or columns found in the provided models. Only basic
// additions are handled.
func GenerateMigrations(db *gorm.DB, models []interface{}, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	var upStmts []string
	var downStmts []string
	for _, m := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return err
		}

		exists := db.Migrator().HasTable(m)
		if !exists {
			dry := db.Session(&gorm.Session{DryRun: true})
			if err := dry.Migrator().CreateTable(m); err != nil {
				return err
			}
			upStmts = append(upStmts, dry.Statement.SQL.String())

			drop := db.Session(&gorm.Session{DryRun: true})
			if err := drop.Migrator().DropTable(m); err != nil {
				return err
			}
			downStmts = append([]string{drop.Statement.SQL.String()}, downStmts...)
			continue
		}

		cols, err := db.Migrator().ColumnTypes(m)
		if err != nil {
			return err
		}
		existing := map[string]struct{}{}
		for _, c := range cols {
			existing[strings.ToLower(c.Name())] = struct{}{}
		}
		for _, f := range stmt.Schema.Fields {
			name := strings.ToLower(f.DBName)
			if _, ok := existing[name]; ok {
				continue
			}
			dry := db.Session(&gorm.Session{DryRun: true})
			if err := dry.Migrator().AddColumn(m, f.DBName); err != nil {
				return err
			}
			upStmts = append(upStmts, dry.Statement.SQL.String())

			drop := db.Session(&gorm.Session{DryRun: true})
			if err := drop.Migrator().DropColumn(m, f.DBName); err != nil {
				return err
			}
			downStmts = append([]string{drop.Statement.SQL.String()}, downStmts...)
		}
	}

	if len(upStmts) == 0 {
		return nil
	}

	name := time.Now().Format("20060102150405") + "_auto"
	upFile := filepath.Join(dir, name+".up.sql")
	downFile := filepath.Join(dir, name+".down.sql")
	if err := os.WriteFile(upFile, []byte(strings.Join(upStmts, "\n")), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(downFile, []byte(strings.Join(downStmts, "\n")), 0o644); err != nil {
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
