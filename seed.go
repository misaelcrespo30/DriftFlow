package schemaflow

import (
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

// Seed executes all SQL seed files in the given directory. Files are executed
// in alphabetical order and must have a `.seed.sql` suffix.
func Seed(db *gorm.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.seed.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := db.Exec(string(b)).Error; err != nil {
			return err
		}
	}
	return nil
}
