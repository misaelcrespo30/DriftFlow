package driftflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

func dummyValue(t reflect.Type, idx int, base time.Time) interface{} {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return base.Add(time.Duration(idx) * time.Hour)
	}
	switch t.Kind() {
	case reflect.Bool:
		return idx%2 == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return idx + 1
	case reflect.Float32, reflect.Float64:
		return float64(idx + 1)
	case reflect.String:
		return fmt.Sprintf("value %d", idx+1)
	default:
		return reflect.Zero(t).Interface()
	}
}

// GenerateSeedTemplates writes JSON seed files with dummy data for the provided
// models into dir. Each file contains an array of 10 objects and will be
// overwritten if it already exists.
func GenerateSeedTemplates(models []interface{}, dir string) error {
	return GenerateSeedTemplatesWithData(models, dir, nil)
}

// GenerateSeedTemplatesWithData is like GenerateSeedTemplates but allows providing
// custom generator functions for field values. The map key should match the JSON
// field name. If no generator is found for a field, a zero value is used.
func GenerateSeedTemplatesWithData(models []interface{}, dir string, gens map[string]func() interface{}) error {
	if strings.TrimSpace(dir) == "" {
		dir = os.Getenv("SEED_GEN_DIR")
		if strings.TrimSpace(dir) == "" {
			dir = "seed"
			fmt.Println("No se definió 'SEED_DIR', se usará ruta por defecto: ./seed")
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := time.Now()
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

		// Skip generation when the seed file already exists
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		objs := make([]*orderedMap, 10)
		for i := 0; i < 10; i++ {
			obj := newOrderedMap()
			for j := 0; j < t.NumField(); j++ {
				f := t.Field(j)
				if !f.IsExported() {
					continue
				}
				if f.Anonymous && f.Type.PkgPath() == "gorm.io/gorm" && f.Type.Name() == "Model" {
					continue
				}
				gtag := f.Tag.Get("gorm")
				if gtag == "-" || strings.Contains(gtag, "->") {
					continue
				}
				tag := f.Tag.Get("json")
				if strings.Split(tag, ",")[0] == "-" {
					continue
				}
				name := strings.Split(tag, ",")[0]
				if name == "" {
					name = strings.ToLower(f.Name)
				}
				if gens != nil {
					if fn, ok := gens[name]; ok {
						obj.set(name, fn())
						continue
					}
				}
				obj.set(name, dummyValue(f.Type, i, base))
			}
			objs[i] = obj
		}

		b, err := json.MarshalIndent(objs, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}
