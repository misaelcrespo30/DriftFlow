package driftflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func zeroValue(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.Bool:
		return false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 0
	case reflect.Float32, reflect.Float64:
		return 0
	case reflect.String:
		return ""
	default:
		return nil
	}
}

// GenerateSeedTemplates writes JSON seed templates for the provided models into dir.
// Existing files are left untouched.
func GenerateSeedTemplates(models []interface{}, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		file := strings.ToLower(t.Name()) + ".json"
		path := filepath.Join(dir, file)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		obj := map[string]interface{}{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name := strings.Split(tag, ",")[0]
			if name == "" {
				name = strings.ToLower(f.Name)
			}
			obj[name] = zeroValue(f.Type)
		}
		b, err := json.MarshalIndent([]map[string]interface{}{obj}, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
