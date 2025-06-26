package driftflow

import (
	"path/filepath"
	"reflect"
	"strings"

	"gorm.io/gorm"

	"github.com/misaelcrespo30/DriftFlow/config"
)

// Seeder defines a type that can seed itself using a JSON file.
type Seeder interface {
	Seed(db *gorm.DB, filePath string) error
}

// Seed executes the Seed method of each provided Seeder using files in dir.
// The file name is derived from the seeder type name in lower case with a .json
// extension (e.g. Bookmark -> bookmark.json).
func Seed(db *gorm.DB, dir string, seeders []Seeder) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	for _, s := range seeders {
		t := reflect.TypeOf(s)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		file := strings.ToLower(t.Name()) + ".json"
		path := filepath.Join(dir, file)
		if err := s.Seed(db, path); err != nil {
			return err
		}
		LogAuditEvent(db, file, "seed")
	}
	return nil
}
