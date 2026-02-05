package driftflow

import (
	"crypto/sha256"
	"encoding/hex"
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
	Batch     int       `gorm:"not null"`
	Checksum  string    `gorm:"size:64;not null"` // sha256
	AppliedAt time.Time `gorm:"autoCreateTime" json:"applied_at"`
}

func (SchemaMigration) TableName() string {
	return "migrations_history"
}

// ensureMigrationsTable creates the schema_migrations table if it does not exist.
func ensureMigrationsTable(db *gorm.DB) error {
	return db.AutoMigrate(&SchemaMigration{})
}

func ensureManifestIntegrity(dir string) error {
	manifestPath := filepath.Join(dir, "manifest.lock.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	migrated, err := migrateManifest(dir, manifest)
	if err != nil {
		return err
	}
	if migrated {
		return fmt.Errorf("manifest.lock.json needs migration; run driftflow generate to update it")
	}

	issues, err := validateManifest(dir, manifest)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("manifest validation failed:\n")
	for _, issue := range issues {
		sb.WriteString(" - ")
		sb.WriteString(string(issue.Type))
		if issue.Migration != "" {
			sb.WriteString(": ")
			sb.WriteString(issue.Migration)
		}
		if issue.File != "" {
			sb.WriteString(" (")
			sb.WriteString(issue.File)
			sb.WriteString(")")
		}
		if issue.Detail != "" {
			sb.WriteString(" - ")
			sb.WriteString(issue.Detail)
		}
		sb.WriteString("\n")
	}
	return fmt.Errorf(strings.TrimSpace(sb.String()))
}

// readMigrationFiles returns the migration files sorted by name.
func readMigrationFiles(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		return migrationVersionFromFilename(files[i]) < migrationVersionFromFilename(files[j])
	})
	return files, nil
}

func migrationVersion(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".sql")
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

// toSnakeWithInitialisms converts CamelCase names to snake_case and keeps
// common initialisms grouped (e.g. ProjectID -> project_id, URLValue -> url_value).
func toSnakeWithInitialisms(s string) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) == 0 {
		return ""
	}

	var out []rune
	for i, r := range runes {
		if i > 0 {
			prev := runes[i-1]
			var next rune
			hasNext := i+1 < len(runes)
			if hasNext {
				next = runes[i+1]
			}

			if (unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev))) ||
				(unicode.IsUpper(r) && unicode.IsUpper(prev) && hasNext && unicode.IsLower(next)) ||
				(unicode.IsDigit(r) && unicode.IsLetter(prev)) ||
				(unicode.IsLetter(r) && unicode.IsDigit(prev)) {
				out = append(out, '_')
			}
		}
		out = append(out, unicode.ToLower(r))
	}

	return string(out)
}

// toSnakeCase keeps backward compatibility while delegating to the improved
// implementation that correctly handles initialisms.
func toSnakeCase(s string) string {
	return toSnakeWithInitialisms(s)
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

func isDatatypesJSON(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.PkgPath() == "gorm.io/datatypes" && (t.Name() == "JSON" || t.Name() == "JSONMap")
}

// sqlTypeOf provides a simple mapping from Go types to SQL types.
func sqlTypeOf(t reflect.Type) string {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct && t.PkgPath() == "gorm.io/gorm" && t.Name() == "DeletedAt" {
		return "timestamp"
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
func columnDef(f reflect.StructField, engine string, hasSoftDelete bool) (string, string) {
	tag := f.Tag.Get("gorm")
	isDeletedAt := isGormDeletedAtType(f.Type)
	typ := getTagValue(tag, "type")
	size := getTagValue(tag, "size")
	if typ == "" {
		if isDatatypesJSON(f.Type) {
			if normalizeEngine(engine) == "postgres" {
				typ = "jsonb"
			} else {
				typ = "json"
			}
		} else if size != "" && f.Type.Kind() == reflect.String {
			typ = fmt.Sprintf("varchar(%s)", size)
		} else {
			typ = sqlTypeOf(f.Type)
		}
	}
	if isDeletedAt {
		if typ == "" || !strings.Contains(strings.ToLower(typ), "timestamp") {
			typ = "timestamp"
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
		if normalizeEngine(engine) != "postgres" || !hasSoftDelete {
			parts = append(parts, "unique")
		}
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
/*func Up(db *gorm.DB, dir string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_ = EnsureFieldHistoryTable(db)
	ups, err := readMigrationFiles(dir)
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
}*/

func Up(db *gorm.DB, dir string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureManifestIntegrity(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	ups, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}

	// nuevo batch = max(batch)+1
	var lastBatch int
	_ = db.Model(&SchemaMigration{}).
		Select("COALESCE(MAX(batch),0)").
		Scan(&lastBatch).Error
	newBatch := lastBatch + 1

	for _, f := range ups {
		version := migrationVersionFromFilename(f) // ✅ usa filename estable
		upSQL, _, checksum, err := readMigrationFile(f)
		if err != nil {
			return err
		}

		// ya aplicada?
		var m SchemaMigration
		result := db.Where("version = ?", version).Limit(1).Find(&m)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			// ✅ si ya existe, valida checksum (opcional pero recomendado)
			if m.Checksum != checksum {
				return fmt.Errorf("migration modified after applied: %s", version)
			}
			continue
		}

		// ✅ aplicar en transacción
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(upSQL).Error; err != nil {
				return fmt.Errorf("apply %s: %w", f, err)
			}
			rec := SchemaMigration{
				Version:   version,
				Batch:     newBatch,
				Checksum:  checksum,
				AppliedAt: time.Now().UTC(),
			}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func migrationVersionFromFilename(path string) string {
	base := filepath.Base(path)             // 2025_..._create_X_table.sql
	return strings.TrimSuffix(base, ".sql") // 2025_..._create_X_table
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Down rolls back migrations until targetVersion is reached.
func Down(db *gorm.DB, dir string, targetVersion string) error {
	return MigrateTo(db, dir, targetVersion)
}

// DownSteps rolls back the most recent N migrations. If steps is less than 1
// or greater than the number of applied migrations, all applied migrations are
// rolled back.
func DownSteps(db *gorm.DB, dir string, steps int) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureManifestIntegrity(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_ = EnsureFieldHistoryTable(db)
	downs, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	downMap := make(map[string]string)
	var versions []string
	for _, f := range downs {
		version := migrationVersionFromFilename(f)
		downMap[version] = f
		versions = append(versions, version)
	}
	var applied []SchemaMigration
	if err := db.Order("id asc").Find(&applied).Error; err != nil {
		return err
	}
	appliedSet := make(map[string]SchemaMigration, len(applied))
	for _, m := range applied {
		appliedSet[m.Version] = m
	}
	var appliedOrdered []string
	for _, v := range versions {
		if _, ok := appliedSet[v]; ok {
			appliedOrdered = append(appliedOrdered, v)
		}
	}
	if steps < 1 || steps > len(appliedOrdered) {
		steps = len(appliedOrdered)
	}
	for i := 0; i < steps; i++ {
		version := appliedOrdered[len(appliedOrdered)-1-i]
		file, ok := downMap[version]
		if !ok {
			return fmt.Errorf("missing down file for %s", version)
		}
		_, downSQL, checksum, err := readMigrationFile(file)
		if err != nil {
			return err
		}
		if appliedSet[version].Checksum != checksum {
			return fmt.Errorf("migration modified after applied: %s", version)
		}
		if err := db.Exec(downSQL).Error; err != nil {
			return fmt.Errorf("revert %s: %w", file, err)
		}
		if err := removeMigration(db, version); err != nil {
			return err
		}
		LogAuditEvent(db, version, "rollback")
	}
	return nil
}

// MigrateTo applies or rolls back migrations until the target version is reached.
func MigrateTo(db *gorm.DB, dir string, targetVersion string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	if err := ensureManifestIntegrity(dir); err != nil {
		return err
	}
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	_ = EnsureAuditTable(db)
	_ = EnsureFieldHistoryTable(db)

	files, err := readMigrationFiles(dir)
	if err != nil {
		return err
	}
	versionToFile := make(map[string]string, len(files))
	var versions []string
	for _, f := range files {
		version := migrationVersionFromFilename(f)
		versionToFile[version] = f
		versions = append(versions, version)
	}

	targetIndex := -1
	for i, v := range versions {
		if v == targetVersion {
			targetIndex = i
			break
		}
	}
	if targetIndex == -1 {
		return fmt.Errorf("target version not found: %s", targetVersion)
	}

	var applied []SchemaMigration
	if err := db.Order("id asc").Find(&applied).Error; err != nil {
		return err
	}
	appliedSet := make(map[string]SchemaMigration, len(applied))
	for _, m := range applied {
		if _, ok := versionToFile[m.Version]; !ok {
			return fmt.Errorf("applied migration missing from disk: %s", m.Version)
		}
		appliedSet[m.Version] = m
	}

	currentIndex := -1
	seenGap := false
	for i, v := range versions {
		if _, ok := appliedSet[v]; ok {
			if seenGap {
				return fmt.Errorf("applied migrations are not contiguous; missing %s", versions[currentIndex+1])
			}
			currentIndex = i
		} else if currentIndex != -1 {
			seenGap = true
		}
	}

	if currentIndex == targetIndex {
		return nil
	}

	if currentIndex < targetIndex {
		var lastBatch int
		_ = db.Model(&SchemaMigration{}).
			Select("COALESCE(MAX(batch),0)").
			Scan(&lastBatch).Error
		newBatch := lastBatch + 1

		for i := currentIndex + 1; i <= targetIndex; i++ {
			version := versions[i]
			file := versionToFile[version]
			upSQL, _, checksum, err := readMigrationFile(file)
			if err != nil {
				return err
			}

			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec(upSQL).Error; err != nil {
					return fmt.Errorf("apply %s: %w", file, err)
				}
				rec := SchemaMigration{
					Version:   version,
					Batch:     newBatch,
					Checksum:  checksum,
					AppliedAt: time.Now().UTC(),
				}
				if err := tx.Create(&rec).Error; err != nil {
					return err
				}
				return nil
			}); err != nil {
				return err
			}
			LogAuditEvent(db, version, "apply")
		}
		return nil
	}

	for i := currentIndex; i > targetIndex; i-- {
		version := versions[i]
		file := versionToFile[version]
		_, downSQL, checksum, err := readMigrationFile(file)
		if err != nil {
			return err
		}
		applied, ok := appliedSet[version]
		if !ok {
			return fmt.Errorf("applied migration missing from db: %s", version)
		}
		if applied.Checksum != checksum {
			return fmt.Errorf("migration modified after applied: %s", version)
		}
		if err := db.Exec(downSQL).Error; err != nil {
			return fmt.Errorf("revert %s: %w", file, err)
		}
		if err := removeMigration(db, version); err != nil {
			return err
		}
		LogAuditEvent(db, version, "rollback")
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

		if err := writeMigrationFile(dir, name, upSQL, downSQL); err != nil {
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

type relationKind string

const (
	relationNone      relationKind = "none"
	relationHasOne    relationKind = "has_one"
	relationHasMany   relationKind = "has_many"
	relationBelongsTo relationKind = "belongs_to"
)

type relationInfo struct {
	Kind             relationKind
	ForeignKeyColumn string
	ReferencesTable  string
	ReferencesColumn string
	OwnerTable       string
}

func isNavigationField(field reflect.StructField) bool {
	ft := field.Type
	if ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}

	if ft.Kind() == reflect.Slice {
		if ft.Elem().Kind() == reflect.Uint8 {
			return false
		}
		elem := ft.Elem()
		if elem.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}
		return elem.Kind() == reflect.Struct && !isDatatypesJSON(elem)
	}

	if ft.Kind() != reflect.Struct {
		return false
	}

	if ft.PkgPath() == "time" && ft.Name() == "Time" {
		return false
	}
	if isGormDeletedAtType(ft) {
		return false
	}

	return true
}

func inferRelation(model reflect.Type, field reflect.StructField) relationInfo {
	if model.Kind() == reflect.Pointer {
		model = model.Elem()
	}
	if model.Kind() != reflect.Struct || !isNavigationField(field) {
		return relationInfo{Kind: relationNone}
	}

	fieldType := field.Type
	isSlice := false
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() == reflect.Slice {
		isSlice = true
		fieldType = fieldType.Elem()
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
	}
	if fieldType.Kind() != reflect.Struct {
		return relationInfo{Kind: relationNone}
	}

	gtag := field.Tag.Get("gorm")
	fkFieldName := strings.TrimSpace(getTagValue(gtag, "foreignKey"))
	referencesFieldName := strings.TrimSpace(getTagValue(gtag, "references"))
	if referencesFieldName == "" {
		referencesFieldName = "ID"
	}

	resolveColumnForField := func(t reflect.Type, fieldName string) string {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return toSnakeWithInitialisms(fieldName)
		}
		if f, ok := t.FieldByName(fieldName); ok {
			if col := getTagValue(f.Tag.Get("gorm"), "column"); col != "" {
				return col
			}
			return toSnakeWithInitialisms(f.Name)
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if strings.EqualFold(f.Name, fieldName) {
				if col := getTagValue(f.Tag.Get("gorm"), "column"); col != "" {
					return col
				}
				return toSnakeWithInitialisms(f.Name)
			}
		}
		return toSnakeWithInitialisms(fieldName)
	}

	hasScalarFKField := func(t reflect.Type, fieldName string) bool {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return false
		}
		f, ok := t.FieldByName(fieldName)
		if !ok {
			for i := 0; i < t.NumField(); i++ {
				candidate := t.Field(i)
				if strings.EqualFold(candidate.Name, fieldName) {
					f = candidate
					ok = true
					break
				}
			}
		}
		if !ok {
			return false
		}

		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		return ft.Kind() != reflect.Struct && ft.Kind() != reflect.Slice && ft.Kind() != reflect.Array && ft.Kind() != reflect.Map
	}

	if isSlice {
		if fkFieldName == "" {
			fkFieldName = model.Name() + "ID"
		}
		return relationInfo{
			Kind:             relationHasMany,
			ForeignKeyColumn: resolveColumnForField(fieldType, fkFieldName),
			ReferencesTable:  gormTableName(model),
			ReferencesColumn: resolveColumnForField(model, referencesFieldName),
			OwnerTable:       gormTableName(fieldType),
		}
	}

	belongsTo := getTagValue(gtag, "belongsTo") != ""
	if fkFieldName == "" {
		fkFieldName = field.Name + "ID"
	}
	if hasScalarFKField(model, fkFieldName) {
		belongsTo = true
	}

	if belongsTo {
		return relationInfo{
			Kind:             relationBelongsTo,
			ForeignKeyColumn: resolveColumnForField(model, fkFieldName),
			ReferencesTable:  gormTableName(fieldType),
			ReferencesColumn: resolveColumnForField(fieldType, referencesFieldName),
			OwnerTable:       gormTableName(model),
		}
	}

	if fkFieldName == "" {
		fkFieldName = model.Name() + "ID"
	}
	return relationInfo{
		Kind:             relationHasOne,
		ForeignKeyColumn: resolveColumnForField(fieldType, fkFieldName),
		ReferencesTable:  gormTableName(model),
		ReferencesColumn: resolveColumnForField(model, referencesFieldName),
		OwnerTable:       gormTableName(fieldType),
	}
}

func buildModelSchema(models []interface{}, engine string) (schemaInfo, map[string][]string, map[string]tableInfo, map[string][]foreignKeyInfo, map[string][]IndexDefinition, error) {
	s := make(schemaInfo)
	orderMap := make(map[string][]string)
	defMap := make(map[string]tableInfo)
	fkMap := make(map[string][]foreignKeyInfo)
	idxMap := make(map[string][]IndexDefinition)

	var (
		collectFields func(reflect.Type, tableInfo, tableInfo, *[]string, string)
		hasGormModel  bool
		hasSoftDelete bool
		deletedAtCol  string
		indexPlans    map[string]*indexPlan
		orderCounter  int
	)

	// Helpers
	/*isTimeType := func(t reflect.Type) bool {
		// Covers time.Time and also types named "Time" in package "time"
		return t.Kind() == reflect.Struct && t.PkgPath() == "time" && t.Name() == "Time"
	}*/

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

			gtag := f.Tag.Get("gorm")

			if isNavigationField(f) {
				rel := inferRelation(t, f)
				if rel.Kind != relationNone && rel.ForeignKeyColumn != "" && rel.OwnerTable != "" && rel.ReferencesTable != "" {
					fkMap[rel.OwnerTable] = append(fkMap[rel.OwnerTable], foreignKeyInfo{
						Column:    rel.ForeignKeyColumn,
						RefTable:  rel.ReferencesTable,
						RefColumn: rel.ReferencesColumn,
					})
				}
				continue
			}

			// ✅ Normal field -> create column
			name := getTagValue(gtag, "column")
			if name == "" {
				name = toSnakeCase(f.Name)
			}

			*order = append(*order, name)

			base, full := columnDef(f, engine, hasSoftDelete)
			cols[name] = base
			defs[name] = full

			for _, tag := range parseIndexTags(gtag) {
				if tag.Kind == indexKindUniqueConstraint && !(normalizeEngine(engine) == "postgres" && hasSoftDelete) {
					continue
				}
				addIndexPlan(indexPlans, tag, name, orderCounter)
				orderCounter++
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
		hasSoftDelete, deletedAtCol = modelDeletedAtInfo(t)
		cols := make(tableInfo)
		defs := make(tableInfo)
		var order []string
		indexPlans = make(map[string]*indexPlan)
		orderCounter = 0

		hasGormModel = false
		collectFields(t, cols, defs, &order, table)

		// GORM model defaults
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
			idxMap[table] = buildIndexDefinitions(table, indexPlans, deletedAtCol, engine, hasSoftDelete)
		}
	}

	return s, orderMap, defMap, fkMap, idxMap, nil
}

func createTableSQL(table string, cols tableInfo, order []string, fks []foreignKeyInfo, engine string) string {
	var defs []string

	// Build column defs
	addCol := func(col string, typ string) {
		defs = append(defs, fmt.Sprintf("%s %s", quoteIdent(engine, col), typ))
	}

	if len(order) == 0 {
		for c, t := range cols {
			addCol(c, t)
		}
	} else {
		// First respect order slice
		seen := make(map[string]bool, len(cols))

		for _, c := range order {
			if t, ok := cols[c]; ok {
				addCol(c, t)
				seen[c] = true
			}
		}
		// Then append remaining columns not present in order
		for c, t := range cols {
			if !seen[c] {
				addCol(c, t)
			}
		}
	}

	// Foreign keys
	for _, fk := range fks {
		defs = append(defs, fmt.Sprintf(
			"FOREIGN KEY (%s) REFERENCES %s(%s)",
			quoteIdent(engine, fk.Column),
			quoteIdent(engine, fk.RefTable),
			quoteIdent(engine, fk.RefColumn),
		))
	}

	return fmt.Sprintf("CREATE TABLE %s (\n  %s\n);", quoteIdent(engine, table), strings.Join(defs, ",\n  "))
}

func fileContent(path string) (string, error) {
	upSQL, _, err := readMigrationSections(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(upSQL), nil
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
	createPattern := filepath.Join(dir, fmt.Sprintf("*_create_%s_table.sql", table))
	addPattern := filepath.Join(dir, fmt.Sprintf("*_add_fields_to_%s_table.sql", table))
	removePattern := filepath.Join(dir, fmt.Sprintf("*_remove_fields_from_%s_table.sql", table))

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
