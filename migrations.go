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
	"unicode"

	"gorm.io/gorm/schema"

	"gorm.io/gorm"

	"github.com/misaelcrespo30/DriftFlow/config"
)

// SchemaMigration represents a row in the schema_migrations table.
type SchemaMigration struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Version   string    `gorm:"uniqueIndex" json:"version"`
	AppliedAt time.Time `gorm:"autoCreateTime" json:"applied_at"`
}

func (SchemaMigration) TableName() string {
	return "migrations_history"
}

// ensureMigrationsTable creates the schema_migrations table if it does not exist.
func ensureMigrationsTable(db *gorm.DB) error {
	return db.AutoMigrate(&SchemaMigration{})
}

// readMigrationFiles returns the up and down migration files sorted by name.
func readMigrationFiles(dir string) (ups, downs []string, err error) {
	ups, err = filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return nil, nil, err
	}
	downs, err = filepath.Glob(filepath.Join(dir, "*.down.sql"))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(ups)
	sort.Strings(downs)
	return ups, downs, nil
}

func migrationVersion(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".up.sql")
	base = strings.TrimSuffix(base, ".down.sql")
	return base
}

// recordMigration inserts a migration record if it doesn't already exist.
func recordMigration(db *gorm.DB, version string) error {
	m := SchemaMigration{Version: version}
	return db.Where("version = ?", version).FirstOrCreate(&m).Error
}

// removeMigration removes a migration record by version.
func removeMigration(db *gorm.DB, version string) error {
	return db.Where("version = ?", version).Delete(&SchemaMigration{}).Error
}

// toSnakeCase converts CamelCase names to snake_case.
func toSnakeCase(s string) string {
	var out []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}

// gormTableName returns the table name for the given type following
// GORM's default naming strategy and honoring a TableName method if present.
func gormTableName(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	if tabler, ok := reflect.New(t).Interface().(interface{ TableName() string }); ok {
		return tabler.TableName()
	}
	return schema.NamingStrategy{}.TableName(t.Name())
}

// getTagValue extracts a value for key from a gorm struct tag.
func getTagValue(tag, key string) string {
	parts := strings.Split(tag, ";")
	prefix := key + ":"
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if strings.HasPrefix(t, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(t, prefix))
		}
	}
	return ""
}

// sqlTypeOf provides a simple mapping from Go types to SQL types.
func sqlTypeOf(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.PkgPath() == "time" && t.Name() == "Time" {
		return "timestamp"
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if t.Kind() == reflect.Int64 {
			return "bigint"
		}
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if t.Kind() == reflect.Uint64 {
			return "bigint"
		}
		return "integer"
	case reflect.Float32:
		return "real"
	case reflect.Float64:
		return "double precision"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "text"
	default:
		return "text"
	}
}

// columnDef returns the basic type for diffing and the full column definition
// string including common GORM decorators like primaryKey or autoIncrement.
func columnDef(f reflect.StructField) (string, string) {
	tag := f.Tag.Get("gorm")
	typ := getTagValue(tag, "type")
	size := getTagValue(tag, "size")
	if typ == "" {
		if size != "" && f.Type.Kind() == reflect.String {
			typ = fmt.Sprintf("varchar(%s)", size)
		} else {
			typ = sqlTypeOf(f.Type)
		}
	}
	base := typ
	full := typ
	lowTag := strings.ToLower(tag)

	if strings.Contains(lowTag, "autoincrement") {
		if typ == "integer" {
			full = "serial"
		} else if typ == "bigint" {
			full = "bigserial"
		}
	}

	parts := []string{full}

	if strings.Contains(lowTag, "primarykey") {
		parts = append(parts, "primary key")
	}
	if strings.Contains(lowTag, "autoincrement") && full == typ {
		parts = append(parts, "auto_increment")
	}
	if strings.Contains(lowTag, "not null") {
		parts = append(parts, "not null")
	}
	if strings.Contains(lowTag, "uniqueindex") || strings.Contains(lowTag, "unique") {
		parts = append(parts, "unique")
	}
	if defVal := getTagValue(tag, "default"); defVal != "" {
		parts = append(parts, "default "+defVal)
	}

	return base, strings.Join(parts, " ")
}

// modelsSchema builds a schemaInfo map from the provided models.
func modelsSchema(db *gorm.DB, models []interface{}) (schemaInfo, error) {
	s := make(schemaInfo)
	for _, m := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return nil, err
		}
		cols := make(tableInfo)
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" || f.IgnoreMigration {
				continue
			}
			name := f.DBName
			if name == "" {
				name = toSnakeCase(f.Name)
			}
			cols[name] = sqlTypeOf(f.FieldType)
		}
		if len(cols) > 0 {
			s[stmt.Schema.Table] = cols
		}
	}
	return s, nil
}

// Up applies all pending migrations found in dir.
func Up(db *gorm.DB, dir string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_ = EnsureFieldHistoryTable(db)
	ups, _, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	for _, f := range ups {
		version := migrationVersion(f)
		var count int64
		if err := db.Model(&SchemaMigration{}).Where("version = ?", version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("apply %s: %w", f, err)
		}
		if err := recordMigration(db, version); err != nil {
			return err
		}
		LogAuditEvent(db, version, "apply")
	}
	return nil
}

// Down rolls back migrations until targetVersion is reached.
func Down(db *gorm.DB, dir string, targetVersion string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_ = EnsureFieldHistoryTable(db)
	_, downs, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	downMap := make(map[string]string)
	for _, f := range downs {
		downMap[migrationVersion(f)] = f
	}
	var applied []SchemaMigration
	if err := db.Order("id desc").Find(&applied).Error; err != nil {
		return err
	}
	for _, m := range applied {
		if m.Version == targetVersion {
			break
		}
		file, ok := downMap[m.Version]
		if !ok {
			return fmt.Errorf("missing down file for %s", m.Version)
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("revert %s: %w", file, err)
		}
		if err := removeMigration(db, m.Version); err != nil {
			return err
		}
		LogAuditEvent(db, m.Version, "rollback")
	}
	return nil
}

// DownSteps rolls back the most recent N migrations. If steps is less than 1
// or greater than the number of applied migrations, all applied migrations are
// rolled back.
func DownSteps(db *gorm.DB, dir string, steps int) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_ = EnsureFieldHistoryTable(db)
	_, downs, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	downMap := make(map[string]string)
	for _, f := range downs {
		downMap[migrationVersion(f)] = f
	}
	var applied []SchemaMigration
	if err := db.Order("id desc").Find(&applied).Error; err != nil {
		return err
	}
	if steps < 1 || steps > len(applied) {
		steps = len(applied)
	}
	for i := 0; i < steps; i++ {
		m := applied[i]
		file, ok := downMap[m.Version]
		if !ok {
			return fmt.Errorf("missing down file for %s", m.Version)
		}
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("revert %s: %w", file, err)
		}
		if err := removeMigration(db, m.Version); err != nil {
			return err
		}
		LogAuditEvent(db, m.Version, "rollback")
	}
	return nil
}

// GenerateMigrations is a placeholder for automatic generation.
// GenerateMigrations inspects the database schema and writes migration files
// for any new tables or columns found in the provided models. Only basic
// additions are handled.
func GenerateMigrations(db *gorm.DB, models []interface{}, dir string) error {
	fmt.Printf("GenerateMigrations dir=%s\n", dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := EnsureFieldHistoryTable(db); err != nil {
		return err
	}

	dbSchema, err := schemaMap(db)
	if err != nil {
		return err
	}

	modelSchema, err := modelsSchema(db, models)
	if err != nil {
		return err
	}

	diffs := diffSchemas(dbSchema, modelSchema)
	if len(diffs) == 0 {
		return nil
	}

	type alter struct {
		col  string
		from string
		to   string
	}
	type change struct {
		create bool
		drop   bool
		add    map[string]string
		remove map[string]string
		alters []alter
	}

	tblChanges := make(map[string]*change)
	getChange := func(tbl string) *change {
		c, ok := tblChanges[tbl]
		if !ok {
			c = &change{add: map[string]string{}, remove: map[string]string{}}
			tblChanges[tbl] = c
		}
		return c
	}

	for _, d := range diffs {
		switch {
		case strings.HasPrefix(d, "[+] table "):
			tbl := strings.TrimPrefix(d, "[+] table ")
			c := getChange(tbl)
			c.create = true
		case strings.HasPrefix(d, "[-] table "):
			tbl := strings.TrimPrefix(d, "[-] table ")
			c := getChange(tbl)
			c.drop = true
		case strings.HasPrefix(d, "[+] column "):
			rest := strings.TrimPrefix(d, "[+] column ")
			parts := strings.Split(rest, ".")
			tbl, col := parts[0], parts[1]
			c := getChange(tbl)
			c.add[col] = modelSchema[tbl][col]
		case strings.HasPrefix(d, "[-] column "):
			rest := strings.TrimPrefix(d, "[-] column ")
			parts := strings.Split(rest, ".")
			tbl, col := parts[0], parts[1]
			c := getChange(tbl)
			c.remove[col] = dbSchema[tbl][col]
		case strings.HasPrefix(d, "[~] column "):
			rest := strings.TrimPrefix(d, "[~] column ")
			parts := strings.Split(rest, " ")
			tblCol := parts[0]
			fromType := parts[1]
			toType := parts[3]
			tp := strings.Split(tblCol, ".")
			tbl, col := tp[0], tp[1]
			c := getChange(tbl)
			c.alters = append(c.alters, alter{col: col, from: fromType, to: toType})
		}
	}

	if len(tblChanges) == 0 {
		return nil
	}

	now := time.Now().UTC()
	i := 0
	for tbl, ch := range tblChanges {
		ts := now.Add(time.Duration(i) * time.Second).Format("2006_01_02_150405")
		i++
		var name, upSQL, downSQL string

		switch {
		case ch.create:
			cols := modelSchema[tbl]
			var defs []string
			for c, t := range cols {
				defs = append(defs, fmt.Sprintf("%s %s", c, t))
			}
			sort.Strings(defs)
			upSQL = fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", tbl, strings.Join(defs, ",\n  "))
			downSQL = fmt.Sprintf("DROP TABLE %s;", tbl)
			name = fmt.Sprintf("%s_create_%s_table", ts, tbl)

		case ch.drop:
			cols := dbSchema[tbl]
			var defs []string
			for c, t := range cols {
				defs = append(defs, fmt.Sprintf("%s %s", c, t))
			}
			sort.Strings(defs)
			upSQL = fmt.Sprintf("DROP TABLE %s;", tbl)
			downSQL = fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", tbl, strings.Join(defs, ",\n  "))
			name = fmt.Sprintf("%s_drop_%s_table", ts, tbl)

		default:
			var upParts, downParts []string
			for col, typ := range ch.add {
				upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tbl, col, typ))
				downParts = append([]string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tbl, col)}, downParts...)
			}
			for col, typ := range ch.remove {
				upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tbl, col))
				downParts = append([]string{fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tbl, col, typ)}, downParts...)
			}
			for _, a := range ch.alters {
				upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", tbl, a.col, a.to))
				downParts = append([]string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", tbl, a.col, a.from)}, downParts...)
			}
			if len(upParts) == 0 {
				continue
			}
			upSQL = strings.Join(upParts, "\n")
			downSQL = strings.Join(downParts, "\n")
			name = fmt.Sprintf("%s_alter_%s_table", ts, tbl)
		}

		if name == "" {
			continue
		}

		upPath := filepath.Join(dir, name+".up.sql")
		downPath := filepath.Join(dir, name+".down.sql")

		if err := os.WriteFile(upPath, []byte(upSQL+"\n"), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(downPath, []byte(downSQL+"\n"), 0o644); err != nil {
			return err
		}

		for col, typ := range ch.add {
			logFieldAdd(db, name, tbl, col, typ)
		}
		for col, typ := range ch.remove {
			logFieldRemove(db, name, tbl, col, typ)
		}
		for _, a := range ch.alters {
			logFieldAlter(db, name, tbl, a.col, a.from, a.to)
		}
		if ch.create {
			for col, typ := range modelSchema[tbl] {
				logFieldAdd(db, name, tbl, col, typ)
			}
		}
		if ch.drop {
			for col, typ := range dbSchema[tbl] {
				logFieldRemove(db, name, tbl, col, typ)
			}
		}
	}

	return nil
}

// Migrate generates migrations from the given models and then applies all
// pending migration files.
func Migrate(db *gorm.DB, dir string, models []interface{}) error {
	if err := GenerateMigrations(db, models, dir); err != nil {
		return err
	}
	return Up(db, dir)
}

// buildModelSchema loads the schema info from struct models.
type foreignKeyInfo struct {
	Column    string
	RefTable  string
	RefColumn string
}

func buildModelSchema(models []interface{}) (schemaInfo, map[string][]string, map[string]tableInfo, map[string][]foreignKeyInfo, error) {
	s := make(schemaInfo)
	orderMap := make(map[string][]string)
	defMap := make(map[string]tableInfo)
	fkMap := make(map[string][]foreignKeyInfo)

	var (
		collectFields func(reflect.Type, tableInfo, tableInfo, *[]string, string)
		hasGormModel  bool
	)

	collectFields = func(t reflect.Type, cols tableInfo, defs tableInfo, order *[]string, tbl string) {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() || f.Tag.Get("gorm") == "-" {
				continue
			}
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			// Handle embedded structs
			if f.Anonymous && ft.Kind() == reflect.Struct {
				if ft.PkgPath() == "gorm.io/gorm" && ft.Name() == "Model" {
					hasGormModel = true
					continue
				}
				collectFields(ft, cols, defs, order, tbl)
				continue
			}
			name := getTagValue(f.Tag.Get("gorm"), "column")
			if name == "" {
				name = toSnakeCase(f.Name)
			}
			*order = append(*order, name)
			base, full := columnDef(f)
			cols[name] = base
			defs[name] = full

			// Foreign key relations
			if (ft.Kind() == reflect.Struct) && !f.Anonymous {
				fkField := getTagValue(f.Tag.Get("gorm"), "foreignKey")
				if fkField != "" {
					refTbl := gormTableName(ft)
					refCol := getTagValue(f.Tag.Get("gorm"), "references")
					if refCol == "" {
						refCol = "id"
					} else {
						refCol = toSnakeCase(refCol)
					}

					fkFields := strings.Split(fkField, ",")
					for _, fk := range fkFields {
						fk = strings.TrimSpace(fk)
						fkCol := toSnakeCase(fk)
						if fld, ok := t.FieldByName(fk); ok {
							colName := getTagValue(fld.Tag.Get("gorm"), "column")
							if colName != "" {
								fkCol = colName
							}
						}
						fkMap[tbl] = append(fkMap[tbl], foreignKeyInfo{Column: fkCol, RefTable: refTbl, RefColumn: refCol})
					}
				}
			}
		}
	}

	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		table := gormTableName(t)
		cols := make(tableInfo)
		defs := make(tableInfo)
		var order []string
		hasGormModel = false
		collectFields(t, cols, defs, &order, table)

		if hasGormModel {
			if _, ok := cols["id"]; !ok {
				cols["id"] = sqlTypeOf(reflect.TypeOf(uint(0)))
				defs["id"] = "serial primary key"
			}
			if _, ok := cols["created_at"]; !ok {
				cols["created_at"] = sqlTypeOf(reflect.TypeOf(time.Time{}))
				defs["created_at"] = "timestamp"
			}
			if _, ok := cols["updated_at"]; !ok {
				cols["updated_at"] = sqlTypeOf(reflect.TypeOf(time.Time{}))
				defs["updated_at"] = "timestamp"
			}
			if _, ok := cols["deleted_at"]; !ok {
				cols["deleted_at"] = sqlTypeOf(reflect.TypeOf(time.Time{}))
				defs["deleted_at"] = "timestamp"
			}
		}

		if len(cols) > 0 {
			s[table] = cols
			orderMap[table] = order
			defMap[table] = defs
		}
	}

	return s, orderMap, defMap, fkMap, nil
}

func createTableSQL(table string, cols tableInfo, order []string, fks []foreignKeyInfo) string {
	var defs []string
	if len(order) == 0 {
		for c, t := range cols {
			defs = append(defs, fmt.Sprintf("%s %s", c, t))
		}
	} else {
		for _, c := range order {
			if t, ok := cols[c]; ok {
				defs = append(defs, fmt.Sprintf("%s %s", c, t))
			}
		}
		for c, t := range cols {
			found := false
			for _, oc := range order {
				if c == oc {
					found = true
					break
				}
			}
			if !found {
				defs = append(defs, fmt.Sprintf("%s %s", c, t))
			}
		}
	}
	for _, fk := range fks {
		defs = append(defs, fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)", fk.Column, fk.RefTable, fk.RefColumn))
	}
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

// GenerateModelMigrations compares models with existing migration files and
// writes new migration files for any differences found.
func GenerateModelMigrations(models []interface{}, dir string) error {
	if dir == "" {
		dir = os.Getenv("MIGRATIONS_PATH")
		if dir == "" {
			dir = "migrations"
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	schema, orderMap, defMap, fkMap, err := buildModelSchema(models)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	idx := 0

	for table, cols := range schema {
		existing, err := loadTableState(dir, table)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				ts := now.Add(time.Duration(idx) * time.Second).Format("2006_01_02_150405")
				idx++
				sql := createTableSQL(table, defMap[table], orderMap[table], fkMap[table])
				name := fmt.Sprintf("%s_create_%s_table.up.sql", ts, table)
				if !duplicateContent(dir, fmt.Sprintf("*_create_%s_table.up.sql", table), sql) {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(sql+"\n"), 0o644); err != nil {
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
			for _, c := range orderMap[table] {
				if _, ok := added[c]; ok {
					stmts = append(stmts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, c, defMap[table][c]))
				}
			}
			sql := strings.Join(stmts, "\n")
			pattern := fmt.Sprintf("*_add_fields_to_%s_table.up.sql", table)
			if !duplicateContent(dir, pattern, sql) {
				name := fmt.Sprintf("%s_add_fields_to_%s_table.up.sql", ts, table)
				if err := os.WriteFile(filepath.Join(dir, name), []byte(sql+"\n"), 0o644); err != nil {
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
			if !duplicateContent(dir, pattern, sql) {
				name := fmt.Sprintf("%s_remove_fields_from_%s_table.up.sql", ts, table)
				if err := os.WriteFile(filepath.Join(dir, name), []byte(sql+"\n"), 0o644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
