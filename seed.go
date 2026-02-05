package driftflow

import (
	"encoding/json"
	"errors"
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

var (
	projectSeederRegistry func() []Seeder

	// ErrNoSeederRegistry is returned when the project didn't register any seeders.
	// This should only be surfaced when the user runs seed-related commands.
	ErrNoSeederRegistry = errors.New("no seeders registered (optional): call driftflow.SetSeederRegistry(...) in your project to enable seeds")
)

// SetSeederRegistry registers a function that returns project seeders.
// Optional: if not set, seed commands can be disabled or return ErrNoSeederRegistry.
func SetSeederRegistry(fn func() []Seeder) {
	projectSeederRegistry = fn
}

// HasSeederRegistry indicates if the project registered seeders.
func HasSeederRegistry() bool {
	return projectSeederRegistry != nil
}

// GetSeeders returns the project seeders or nil if none were registered.
func GetSeeders() []Seeder {
	if projectSeederRegistry == nil {
		return nil
	}
	return projectSeederRegistry()
}

// Seed executes the Seed method of each registered Seeder using files in dir.
// File name is derived from the seeder type name in lower case with .seed.json
// (e.g. BookmarkSeeder -> bookmark.seed.json).
// Seed executes the Seed method of each registered Seeder using files in dir.
// File name is derived from the seeder type name in lower case with .seed.json
// (e.g. BookmarkSeeder -> bookmark.seed.json).
func Seed(db *gorm.DB, dir string) error {
	seeders := GetSeeders()
	if len(seeders) == 0 {
		// Important: driftflow should NOT require seeders.
		// Return a clear error only when Seed() is explicitly invoked.
		return ErrNoSeederRegistry
	}

	if err := config.ValidateDir(dir); err != nil {
		return err
	}

	// Ensure audit table exists (use original db; fine).
	_ = EnsureAuditTable(db)

	for _, s := range seeders {
		t := reflect.TypeOf(s)
		if t == nil {
			continue
		}
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		baseName := strings.ToLower(strings.TrimSuffix(t.Name(), "Seeder"))
		file := baseName + ".seed.json"
		path := filepath.Join(dir, file)

		// ✅ Always run each seeder in an isolated, clean session so it doesn't
		// inherit clauses/scopes (e.g., ON CONFLICT) from upstream callers.
		cleanDB := db.Session(&gorm.Session{
			NewDB: true,
		})

		// ✅ Isolate each seed in its own transaction:
		// - prevents partial inserts per seeder
		// - keeps audit record consistent with the insert
		if err := cleanDB.Transaction(func(tx *gorm.DB) error {
			if err := s.Seed(tx, path); err != nil {
				return err
			}
			LogAuditEvent(tx, file, "seed")
			return nil
		}); err != nil {
			return fmt.Errorf("seed %s failed: %w", file, err)
		}
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
		if t == nil {
			continue
		}
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
			return fmt.Errorf("invalid seed json %s: %w", file, err)
		}

		if err := db.Create(slicePtr.Elem().Interface()).Error; err != nil {
			return fmt.Errorf("insert seed %s failed: %w", file, err)
		}

		LogAuditEvent(db, file, "seed")
	}
	return nil
}

/*import (
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
}*/
