package driftflow

import (
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

func latestCreateFile(dir, table string) (string, error) {
	pattern := filepath.Join(dir, fmt.Sprintf("*_create_%s_table.up.sql", table))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return "", fs.ErrNotExist
	}
	return matches[len(matches)-1], nil
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
		genSQL := createTableSQL(table, cols)
		createFile, err := latestCreateFile(migDir, table)
		if err != nil {
			ts := now.Add(time.Duration(idx) * time.Second).Format("2006_01_02_150405")
			idx++
			name := fmt.Sprintf("%s_create_%s_table.up.sql", ts, table)
			if !duplicateContent(migDir, fmt.Sprintf("*_create_%s_table.up.sql", table), genSQL) {
				if err := os.WriteFile(filepath.Join(migDir, name), []byte(genSQL+"\n"), 0o644); err != nil {
					return err
				}
			}
			continue
		}
		oldSQL, err := fileContent(createFile)
		if err != nil {
			return err
		}
		if strings.TrimSpace(oldSQL) == strings.TrimSpace(genSQL) {
			continue
		}
		oldCols := parseCreateColumns(oldSQL)
		added, removed := diffColumns(oldCols, cols)
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
