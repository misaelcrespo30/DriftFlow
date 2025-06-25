package helpers

import (
	"encoding/json"
	"gorm.io/gorm"
	"os"
)

// ReadJSON reads a JSON file at path into v.
func ReadJSON(path string, v interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// GetRandomRecord selects a random record from the given table into dest.
func GetRandomRecord(db *gorm.DB, dest interface{}) interface{} {
	db.Order("RANDOM()").First(dest)
	return dest
}
