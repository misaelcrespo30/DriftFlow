package driftflow

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/misaelcrespo30/DriftFlow/helpers"
)

// buildModelSchema loads the schema info from struct models.
func buildModelSchema(models []interface{}) (schemaInfo, error) {
	s := make(schemaInfo)
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		table := toSnakeCase(t.Name())
		cols := make(tableInfo)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() || f.Tag.Get("gorm") == "-" {
				continue
			}
			name := getTagValue(f.Tag.Get("gorm"), "column")
			if name == "" {
				name = toSnakeCase(f.Name)
			}
			cols[name] = sqlTypeOf(f.Type)
		}
		if len(cols) > 0 {
			s[table] = cols
		}
	}
	return s, nil
}

func createTableSQL(table string, cols tableInfo) string {
	var defs []string
	for c, t := range cols {
		defs = append(defs, fmt.Sprintf("%s %s", c, t))
	}
	sort.Strings(defs)
	return fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", table, strings.Join(defs, ",\n  "))
}

func fileContent(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func parseCreateColumns(sql string) tableInfo {
	start := strings.Index(sql, "(")
	end := strings.LastIndex(sql, ")")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	inner := sql[start+1 : end]
	cols := make(tableInfo)
	for _, line := range strings.Split(inner, ",") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			cols[parts[0]] = parts[1]
		}
	}
	return cols
}

func diffColumns(old, new tableInfo) (added, removed tableInfo) {
	added = make(tableInfo)
	removed = make(tableInfo)
	for c, t := range new {
		if _, ok := old[c]; !ok {
			added[c] = t
		}
	}
	for c, t := range old {
		if _, ok := new[c]; !ok {
			removed[c] = t
		}
	}
	return
}

func parseAddColumns(sql string) tableInfo {
	cols := make(tableInfo)
	for _, line := range strings.Split(sql, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, ";"))
		if line == "" {
			continue
		}
		idx := strings.Index(line, "ADD COLUMN ")
		if idx == -1 {
			continue
		}
		rest := line[idx+len("ADD COLUMN "):]
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			continue
		}
		col := parts[0]
		typ := strings.Join(parts[1:], " ")
		cols[col] = typ
	}
	return cols
}

func parseRemoveColumns(sql string) []string {
	var cols []string
	for _, line := range strings.Split(sql, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, ";"))
		if line == "" {
			continue
		}
		idx := strings.Index(line, "DROP COLUMN ")
		if idx == -1 {
			continue
		}
		rest := line[idx+len("DROP COLUMN "):]
		parts := strings.Fields(rest)
		if len(parts) < 1 {
			continue
		}
		cols = append(cols, parts[0])
	}
	return cols
}

func loadTableState(dir, table string) (tableInfo, error) {
	createPattern := filepath.Join(dir, fmt.Sprintf("*_create_%s_table.up.sql", table))
	addPattern := filepath.Join(dir, fmt.Sprintf("*_add_fields_to_%s_table.up.sql", table))
	removePattern := filepath.Join(dir, fmt.Sprintf("*_remove_fields_from_%s_table.up.sql", table))

	files, err := filepath.Glob(createPattern)
	if err != nil {
		return nil, err
	}
	addFiles, _ := filepath.Glob(addPattern)
	removeFiles, _ := filepath.Glob(removePattern)
	files = append(files, addFiles...)
	files = append(files, removeFiles...)
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fs.ErrNotExist
	}

	cols := make(tableInfo)
	for _, f := range files {
		sql, err := fileContent(f)
		if err != nil {
			return nil, err
		}
		base := filepath.Base(f)
		switch {
		case strings.Contains(base, "_create_"):
			cols = parseCreateColumns(sql)
		case strings.Contains(base, "_add_fields_to_"):
			add := parseAddColumns(sql)
			for c, t := range add {
				cols[c] = t
			}
		case strings.Contains(base, "_remove_fields_from_"):
			rem := parseRemoveColumns(sql)
			for _, c := range rem {
				delete(cols, c)
			}
		}
	}
	return cols, nil
}

func duplicateContent(dir, pattern, sql string) bool {
	files, _ := filepath.Glob(filepath.Join(dir, pattern))
	for _, f := range files {
		b, err := fileContent(f)
		if err == nil && strings.TrimSpace(b) == strings.TrimSpace(sql) {
			return true
		}
	}
	return false
}

// GenerateModelMigrations creates migration files by comparing model structs
// against existing create migrations.
func GenerateModelMigrations() error {
	models, err := helpers.LoadModels()
	if err != nil {
		return err
	}
	migDir := os.Getenv("MIGRATIONS_PATH")
	if migDir == "" {
		migDir = "migrations"
	}
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		return err
	}

	schema, err := buildModelSchema(models)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	idx := 0

	for table, cols := range schema {
		existing, err := loadTableState(migDir, table)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				ts := now.Add(time.Duration(idx) * time.Second).Format("2006_01_02_150405")
				idx++
				sql := createTableSQL(table, cols)
				name := fmt.Sprintf("%s_create_%s_table.up.sql", ts, table)
				if !duplicateContent(migDir, fmt.Sprintf("*_create_%s_table.up.sql", table), sql) {
					if err := os.WriteFile(filepath.Join(migDir, name), []byte(sql+"\n"), 0o644); err != nil {
						return err
					}
				}
				continue
			}
			return err
		}

		added, removed := diffColumns(existing, cols)
		if len(added) > 0 {
			ts := now.Add(time.Duration(idx) * time.Second).Format("2006_01_02_150405")
			idx++
			var stmts []string
			for c, t := range added {
				stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, c, t))
			}
			sql := strings.Join(stmts, "\n")
			pattern := fmt.Sprintf("*_add_fields_to_%s_table.up.sql", table)
			if !duplicateContent(migDir, pattern, sql) {
				name := fmt.Sprintf("%s_add_fields_to_%s_table.up.sql", ts, table)
				if err := os.WriteFile(filepath.Join(migDir, name), []byte(sql+"\n"), 0o644); err != nil {
					return err
				}
			}
		}
		if len(removed) > 0 {
			ts := now.Add(time.Duration(idx) * time.Second).Format("2006_01_02_150405")
			idx++
			var stmts []string
			for c := range removed {
				stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", table, c))
			}
			sql := strings.Join(stmts, "\n")
			pattern := fmt.Sprintf("*_remove_fields_from_%s_table.up.sql", table)
			if !duplicateContent(migDir, pattern, sql) {
				name := fmt.Sprintf("%s_remove_fields_from_%s_table.up.sql", ts, table)
				if err := os.WriteFile(filepath.Join(migDir, name), []byte(sql+"\n"), 0o644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
