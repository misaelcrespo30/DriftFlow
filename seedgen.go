package driftflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/misaelcrespo30/DriftFlow/config"
)

const seedTemplateCount = 10

type seedModelInfo struct {
	modelType     reflect.Type
	primaryJSON   string
	primaryType   reflect.Type
	primaryIsText bool
}

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
		cfg := config.Load()
		dir = cfg.SeedGenDir
		if strings.TrimSpace(dir) == "" {
			dir = "internal/database/data"
			fmt.Println("No se definió 'SEED_GEN_DIR', se usará ruta por defecto:", dir)
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	modelInfos := buildSeedModelInfo(models)
	infoByType := make(map[reflect.Type]seedModelInfo, len(modelInfos))
	primaryIDs := make(map[reflect.Type][]string)
	primaryByJSON := make(map[string]seedModelInfo)
	for _, info := range modelInfos {
		infoByType[info.modelType] = info
		if info.primaryJSON != "" && info.primaryIsText {
			ids := make([]string, seedTemplateCount)
			for i := 0; i < seedTemplateCount; i++ {
				ids[i] = uuid.NewString()
			}
			primaryIDs[info.modelType] = ids
			primaryByJSON[info.primaryJSON] = info
		}
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

		objs := make([]*orderedMap, seedTemplateCount)
		for i := 0; i < seedTemplateCount; i++ {
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
				if id, ok := primaryIDValue(primaryIDs, infoByType, t, name, i); ok {
					obj.set(name, id)
					continue
				}
				if id, ok := foreignIDValue(primaryIDs, primaryByJSON, t, name, i); ok {
					obj.set(name, id)
					continue
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

func buildSeedModelInfo(models []interface{}) []seedModelInfo {
	infos := make([]seedModelInfo, 0, len(models))
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
		info := seedModelInfo{modelType: t}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
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
			if !strings.Contains(gtag, "primaryKey") {
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
			info.primaryJSON = name
			info.primaryType = f.Type
			info.primaryIsText = isTextType(f.Type)
			break
		}
		infos = append(infos, info)
	}
	return infos
}

func primaryIDValue(primaryIDs map[reflect.Type][]string, infoByType map[reflect.Type]seedModelInfo, modelType reflect.Type, jsonName string, idx int) (string, bool) {
	ids, ok := primaryIDs[modelType]
	if !ok {
		return "", false
	}
	if idx < 0 || idx >= len(ids) {
		return "", false
	}
	info, ok := infoByType[modelType]
	if !ok {
		return "", false
	}
	if jsonName == "" {
		return "", false
	}
	if info.primaryJSON != "" && jsonName != info.primaryJSON {
		return "", false
	}
	return ids[idx], true
}

func foreignIDValue(primaryIDs map[reflect.Type][]string, primaryByJSON map[string]seedModelInfo, modelType reflect.Type, jsonName string, idx int) (string, bool) {
	rel, ok := primaryByJSON[jsonName]
	if !ok {
		return "", false
	}
	if rel.modelType == modelType {
		return "", false
	}
	ids, ok := primaryIDs[rel.modelType]
	if !ok || len(ids) == 0 {
		return "", false
	}
	return ids[idx%len(ids)], true
}

func isTextType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.String
}
