package driftflow

import (
	"encoding/json"
	"fmt"
	"os"
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

var projectSeederRegistry func() []Seeder

func SetSeederRegistry(fn func() []Seeder) {
	projectSeederRegistry = fn
}

// Seed executes the Seed method of each provided Seeder using files in dir.
// The file name is derived from the seeder type name in lower case with a .json
// extension (e.g. Bookmark -> bookmark.json).
func Seed(db *gorm.DB, dir string) error {
	if projectSeederRegistry == nil {
		return fmt.Errorf(" No se registró ningún seeder. Usá driftflow.SetSeederRegistry(...) desde tu proyecto")
	}

	seeders := projectSeederRegistry()

	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)

	for _, s := range seeders {
		t := reflect.TypeOf(s)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		baseName := strings.ToLower(strings.TrimSuffix(t.Name(), "Seeder"))
		file := baseName + ".seed.json"
		path := filepath.Join(dir, file)

		if err := s.Seed(db, path); err != nil {
			return err
		}
		LogAuditEvent(db, file, "seed")
	}
	return nil
}

// SeedFromJSON reads seed files for the provided models from dir and inserts
// the records into the database using a bulk Create per file. Files are named
// using the lower-cased struct name with a .seed.json extension.
func SeedFromJSON(db *gorm.DB, dir string, models []interface{}) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		file := strings.ToLower(t.Name()) + ".seed.json"
		path := filepath.Join(dir, file)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		slicePtr := reflect.New(reflect.SliceOf(t))
		if err := json.Unmarshal(data, slicePtr.Interface()); err != nil {
			return err
		}
		if err := db.Create(slicePtr.Elem().Interface()).Error; err != nil {
			return err
		}
		LogAuditEvent(db, file, "seed")
	}
	return nil
}
