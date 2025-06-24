package schemaflow

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
	}
	return nil
}

// Down rolls back migrations until targetVersion is reached.
func Down(db *gorm.DB, dir string, targetVersion string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
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
	}
	return nil
}

// GenerateMigrations is a placeholder for automatic generation.
func GenerateMigrations(models []interface{}, dir string) error {
	// TODO: implement automatic migration generation
	return nil
}
