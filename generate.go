package driftflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type SchemaSnapshot struct {
	Version int                      `json:"version"`
	Tables  map[string]SnapshotTable `json:"tables"`
}

type SnapshotTable struct {
	Columns     map[string]string `json:"columns"`      // col -> full sql def (NOT NULL, default, etc.)
	Order       []string          `json:"order"`        // stable order
	ForeignKeys []foreignKeyInfo  `json:"foreign_keys"` // optional, por ahora solo para create
}

// --------------------
// Manifest (tamper detection)
// --------------------

type ManifestMode int

const (
	ManifestStrict    ManifestMode      = iota // error si hay drift/tamper/untracked
	ManifestRepair                             // re-firma hashes y opcionalmente registra untracked
	IssueHashMismatch ManifestIssueType = "hash_mismatch"
	IssueMissingFile  ManifestIssueType = "missing_file"
	IssueUntracked    ManifestIssueType = "untracked_migration"
	IssueMissingPair  ManifestIssueType = "missing_pair"
)

type GenerateOptions struct {
	Dir                string
	ManifestMode       ManifestMode
	RepairAddUntracked bool // en repair: agrega *.sql fuera del manifest
}

type ManifestLock struct {
	Version    int             `json:"version"`
	Migrations []ManifestEntry `json:"migrations"`
	UpdatedUTC string          `json:"updated_utc,omitempty"`
}

type ManifestEntry struct {
	Name       string `json:"name"` // base name, sin .up.sql/.down.sql
	UpSHA256   string `json:"up_sha256"`
	DownSHA256 string `json:"down_sha256"`
	CreatedUTC string `json:"created_utc"`
}

type ManifestIssueType string

type ManifestIssue struct {
	Type      ManifestIssueType
	Migration string // base name
	File      string // path si aplica
	Detail    string
}

func loadManifest(path string) (*ManifestLock, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ManifestLock{Version: 0, Migrations: []ManifestEntry{}}, nil
		}
		return nil, err
	}
	var m ManifestLock
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("manifest.lock.json invalid: %w", err)
	}
	if m.Migrations == nil {
		m.Migrations = []ManifestEntry{}
	}
	return &m, nil
}

func saveManifest(path string, manifest *ManifestLock) error {
	manifest.UpdatedUTC = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sha256File(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func validateManifest(dir string, manifest *ManifestLock) ([]ManifestIssue, error) {
	entries := make(map[string]ManifestEntry, len(manifest.Migrations))
	for _, e := range manifest.Migrations {
		if e.Name == "" {
			return nil, fmt.Errorf("manifest has empty migration name")
		}
		if _, exists := entries[e.Name]; exists {
			return nil, fmt.Errorf("manifest has duplicate migration name: %s", e.Name)
		}
		entries[e.Name] = e
	}

	var issues []ManifestIssue

	// 1) verify tracked entries hashes
	for name, e := range entries {
		upPath := filepath.Join(dir, name+".up.sql")
		downPath := filepath.Join(dir, name+".down.sql")

		upHash, err := sha256File(upPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				issues = append(issues, ManifestIssue{Type: IssueMissingFile, Migration: name, File: upPath, Detail: "missing up file"})
				continue
			}
			return nil, err
		}

		downHash, err := sha256File(downPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				issues = append(issues, ManifestIssue{Type: IssueMissingFile, Migration: name, File: downPath, Detail: "missing down file"})
				continue
			}
			return nil, err
		}

		if !strings.EqualFold(upHash, e.UpSHA256) {
			issues = append(issues, ManifestIssue{Type: IssueHashMismatch, Migration: name, File: upPath, Detail: "UP hash mismatch"})
		}
		if !strings.EqualFold(downHash, e.DownSHA256) {
			issues = append(issues, ManifestIssue{Type: IssueHashMismatch, Migration: name, File: downPath, Detail: "DOWN hash mismatch"})
		}
	}

	// 2) scan disk and ensure no untracked/missing pairs exist
	ups, _ := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	downs, _ := filepath.Glob(filepath.Join(dir, "*.down.sql"))

	diskBases := map[string]struct{}{}
	addBase := func(p, suffix string) {
		base := strings.TrimSuffix(filepath.Base(p), suffix)
		diskBases[base] = struct{}{}
	}

	for _, p := range ups {
		addBase(p, ".up.sql")
	}
	for _, p := range downs {
		addBase(p, ".down.sql")
	}

	for base := range diskBases {
		up := filepath.Join(dir, base+".up.sql")
		down := filepath.Join(dir, base+".down.sql")

		if _, err := os.Stat(up); err != nil {
			issues = append(issues, ManifestIssue{Type: IssueMissingPair, Migration: base, File: up, Detail: "missing UP file for migration base"})
			continue
		}
		if _, err := os.Stat(down); err != nil {
			issues = append(issues, ManifestIssue{Type: IssueMissingPair, Migration: base, File: down, Detail: "missing DOWN file for migration base"})
			continue
		}

		if _, ok := entries[base]; !ok {
			issues = append(issues, ManifestIssue{Type: IssueUntracked, Migration: base, Detail: "migration exists on disk but is not registered in manifest"})
		}
	}

	return issues, nil
}

func repairManifest(dir string, manifest *ManifestLock, issues []ManifestIssue, addUntracked bool) error {
	byName := map[string]*ManifestEntry{}
	for i := range manifest.Migrations {
		e := &manifest.Migrations[i]
		byName[e.Name] = e
	}

	recalc := func(base string) (string, string, error) {
		upPath := filepath.Join(dir, base+".up.sql")
		downPath := filepath.Join(dir, base+".down.sql")
		upHash, err := sha256File(upPath)
		if err != nil {
			return "", "", err
		}
		downHash, err := sha256File(downPath)
		if err != nil {
			return "", "", err
		}
		return upHash, downHash, nil
	}

	// fix mismatches
	for _, is := range issues {
		if is.Type != IssueHashMismatch {
			continue
		}
		base := is.Migration
		entry := byName[base]
		if entry == nil {
			continue
		}
		upHash, downHash, err := recalc(base)
		if err != nil {
			return err
		}
		entry.UpSHA256 = upHash
		entry.DownSHA256 = downHash
	}

	// add untracked (optional)
	if addUntracked {
		seen := map[string]bool{}
		for _, e := range manifest.Migrations {
			seen[e.Name] = true
		}
		for _, is := range issues {
			if is.Type != IssueUntracked {
				continue
			}
			base := is.Migration
			if seen[base] {
				continue
			}
			upHash, downHash, err := recalc(base)
			if err != nil {
				return err
			}
			manifest.Migrations = append(manifest.Migrations, ManifestEntry{
				Name:       base,
				UpSHA256:   upHash,
				DownSHA256: downHash,
				CreatedUTC: time.Now().UTC().Format(time.RFC3339),
			})
			seen[base] = true
		}
	}

	sort.SliceStable(manifest.Migrations, func(i, j int) bool {
		return manifest.Migrations[i].Name < manifest.Migrations[j].Name
	})

	manifest.Version++
	return nil
}

func appendMigrationToManifest(dir string, manifest *ManifestLock, baseName, createdUTC string) error {
	upPath := filepath.Join(dir, baseName+".up.sql")
	downPath := filepath.Join(dir, baseName+".down.sql")

	upHash, err := sha256File(upPath)
	if err != nil {
		return err
	}
	downHash, err := sha256File(downPath)
	if err != nil {
		return err
	}

	manifest.Migrations = append(manifest.Migrations, ManifestEntry{
		Name:       baseName,
		UpSHA256:   upHash,
		DownSHA256: downHash,
		CreatedUTC: createdUTC,
	})

	sort.SliceStable(manifest.Migrations, func(i, j int) bool {
		return manifest.Migrations[i].Name < manifest.Migrations[j].Name
	})

	return nil
}

// --------------------
// Generate (strict/repair)
// --------------------

// GenerateModelMigrations compares MODELS vs schema.lock.json and writes incremental migration files.
// Default recommended usage:
//   - CI:   ManifestStrict
//   - Dev:  ManifestRepair + RepairAddUntracked=true (si quieres “adoptar” migraciones existentes)
func GenerateModelMigrations(models []interface{}, opts GenerateOptions) error {
	dir := opts.Dir
	if dir == "" {
		dir = os.Getenv("MIG_DIR")
		if dir == "" {
			dir = "migrations"
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// 1) validate/repair manifest
	manifestPath := filepath.Join(dir, "manifest.lock.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}

	issues, err := validateManifest(dir, manifest)
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		if opts.ManifestMode == ManifestStrict {
			first := issues[0]
			return fmt.Errorf("manifest validation failed (%s): %s %s - %s",
				first.Type, first.Migration, first.File, first.Detail)
		}

		if err := repairManifest(dir, manifest, issues, opts.RepairAddUntracked); err != nil {
			return err
		}
		if err := saveManifest(manifestPath, manifest); err != nil {
			return err
		}
	}

	// 2) load snapshot
	lockPath := filepath.Join(dir, "schema.lock.json")
	snap, err := loadSnapshot(lockPath)
	if err != nil {
		return err
	}
	if snap.Tables == nil {
		snap.Tables = map[string]SnapshotTable{}
	}

	// 3) Build schema from models
	schemaMap, orderMap, defMap, fkMap, err := buildModelSchema(models)
	if err != nil {
		return err
	}

	// Tables in stable order based on input models
	tablesInOrder := tablesFromModels(models, schemaMap)

	now := time.Now().UTC()
	seq := 0
	changed := false
	newMigrations := 0

	for _, table := range tablesInOrder {
		modelCols := defMap[table] // col -> full definition
		modelOrder := orderMap[table]
		modelFKs := fkMap[table]

		prev, exists := snap.Tables[table]
		if !exists {
			// CREATE TABLE migration
			name := fmt.Sprintf("%s_create_%s_table", ts(now, seq), table)
			seq++

			up := createTableSQL(table, modelCols, modelOrder, modelFKs)
			down := fmt.Sprintf("DROP TABLE %s;", table)

			if err := writeMigrationPair(dir, name, up, down); err != nil {
				return err
			}
			if err := appendMigrationToManifest(dir, manifest, name, now.Format(time.RFC3339)); err != nil {
				return err
			}
			newMigrations++

			snap.Tables[table] = SnapshotTable{
				Columns:     copyMap(modelCols),
				Order:       append([]string{}, modelOrder...),
				ForeignKeys: append([]foreignKeyInfo{}, modelFKs...),
			}

			changed = true
			continue
		}

		added, removed, altered := diffSnapshot(prev.Columns, modelCols)
		if len(added) == 0 && len(removed) == 0 && len(altered) == 0 {
			continue
		}

		// ALTER TABLE migration (one per table per run)
		name := fmt.Sprintf("%s_alter_%s_table", ts(now, seq), table)
		seq++

		up, down := buildAlterSQL(table, prev.Columns, modelCols, modelOrder, added, removed, altered)
		if strings.TrimSpace(up) == "" {
			continue
		}

		if err := writeMigrationPair(dir, name, up, down); err != nil {
			return err
		}
		if err := appendMigrationToManifest(dir, manifest, name, now.Format(time.RFC3339)); err != nil {
			return err
		}
		newMigrations++

		// Update snapshot state for this table
		prev.Columns = copyMap(modelCols)
		prev.Order = append([]string{}, modelOrder...)
		prev.ForeignKeys = append([]foreignKeyInfo{}, modelFKs...)
		snap.Tables[table] = prev

		changed = true
	}

	// 4) Update snapshot only if something changed
	if changed {
		snap.Version++
		if err := saveSnapshot(lockPath, snap); err != nil {
			return err
		}
	}

	// 5) Update manifest if new migrations were created
	if newMigrations > 0 {
		manifest.Version++
		if err := saveManifest(manifestPath, manifest); err != nil {
			return err
		}
	}

	return nil
}

func tablesFromModels(models []interface{}, schemaMap schemaInfo) []string {
	var tables []string
	seen := map[string]bool{}
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		tbl := gormTableName(t)
		if _, ok := schemaMap[tbl]; ok && !seen[tbl] {
			seen[tbl] = true
			tables = append(tables, tbl)
		}
	}
	return tables
}

func ts(now time.Time, seq int) string {
	return now.Add(time.Duration(seq) * time.Second).Format("2006_01_02_150405")
}

func writeMigrationPair(dir, baseName, upSQL, downSQL string) error {
	upPath := filepath.Join(dir, baseName+".up.sql")
	downPath := filepath.Join(dir, baseName+".down.sql")

	// immutable: never overwrite existing migrations
	if _, err := os.Stat(upPath); err == nil {
		return fmt.Errorf("migration already exists (refusing to overwrite): %s", upPath)
	}
	if _, err := os.Stat(downPath); err == nil {
		return fmt.Errorf("migration already exists (refusing to overwrite): %s", downPath)
	}

	if err := os.WriteFile(upPath, []byte(upSQL+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(downPath, []byte(downSQL+"\n"), 0o644); err != nil {
		return err
	}
	return nil
}

func loadSnapshot(path string) (*SchemaSnapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &SchemaSnapshot{Version: 0, Tables: map[string]SnapshotTable{}}, nil
		}
		return nil, err
	}

	var s SchemaSnapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("schema.lock.json invalid: %w", err)
	}
	if s.Tables == nil {
		s.Tables = map[string]SnapshotTable{}
	}
	return &s, nil
}

func saveSnapshot(path string, snap *SchemaSnapshot) error {
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

/*
type SchemaSnapshot struct {
	Version int                      `json:"version"`
	Tables  map[string]SnapshotTable `json:"tables"`
}

type SnapshotTable struct {
	Columns     map[string]string `json:"columns"`      // col -> full sql def (NOT NULL, default, etc.)
	Order       []string          `json:"order"`        // stable order
	ForeignKeys []foreignKeyInfo  `json:"foreign_keys"` // optional, por ahora solo para create
}

// GenerateModelMigrations compares MODELS vs schema.lock.json
// and writes incremental migration files (snapshot + diffs).
func GenerateModelMigrations(models []interface{}, dir string) error {
	if dir == "" {
		dir = os.Getenv("MIG_DIR")
		if dir == "" {
			dir = "migrations"
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	lockPath := filepath.Join(dir, "schema.lock.json")
	snap, err := loadSnapshot(lockPath)
	if err != nil {
		return err
	}
	if snap.Tables == nil {
		snap.Tables = map[string]SnapshotTable{}
	}

	// Build schema from models
	schemaMap, orderMap, defMap, fkMap, err := buildModelSchema(models)
	if err != nil {
		return err
	}

	// Tables in stable order based on input models
	tablesInOrder := tablesFromModels(models, schemaMap)

	now := time.Now().UTC()
	seq := 0
	changed := false

	for _, table := range tablesInOrder {
		modelCols := defMap[table] // col -> full definition
		modelOrder := orderMap[table]
		modelFKs := fkMap[table]

		prev, exists := snap.Tables[table]
		if !exists {
			// CREATE TABLE migration
			name := fmt.Sprintf("%s_create_%s_table", ts(now, seq), table)
			seq++

			up := createTableSQL(table, modelCols, modelOrder, modelFKs)
			down := fmt.Sprintf("DROP TABLE %s;", table)

			if err := writeMigrationPair(dir, name, up, down); err != nil {
				return err
			}

			snap.Tables[table] = SnapshotTable{
				Columns:     copyMap(modelCols),
				Order:       append([]string{}, modelOrder...),
				ForeignKeys: append([]foreignKeyInfo{}, modelFKs...),
			}

			changed = true
			continue
		}

		added, removed, altered := diffSnapshot(prev.Columns, modelCols)

		// Si no hay cambios reales, no generes nada
		if len(added) == 0 && len(removed) == 0 && len(altered) == 0 {
			continue
		}

		// ALTER TABLE migration (one per table per run)
		name := fmt.Sprintf("%s_alter_%s_table", ts(now, seq), table)
		seq++

		up, down := buildAlterSQL(table, prev.Columns, modelCols, modelOrder, added, removed, altered)

		// Si por alguna razón el SQL quedó vacío, skip
		if strings.TrimSpace(up) == "" {
			continue
		}

		if err := writeMigrationPair(dir, name, up, down); err != nil {
			return err
		}

		// Update snapshot state for this table
		prev.Columns = copyMap(modelCols)
		prev.Order = append([]string{}, modelOrder...)
		prev.ForeignKeys = append([]foreignKeyInfo{}, modelFKs...)
		snap.Tables[table] = prev

		changed = true
	}

	// Update snapshot only if something changed
	if changed {
		snap.Version++
		if err := saveSnapshot(lockPath, snap); err != nil {
			return err
		}
	}

	return nil
}

func tablesFromModels(models []interface{}, schemaMap schemaInfo) []string {
	var tables []string
	seen := map[string]bool{}
	for _, m := range models {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			continue
		}
		tbl := gormTableName(t)
		if _, ok := schemaMap[tbl]; ok && !seen[tbl] {
			seen[tbl] = true
			tables = append(tables, tbl)
		}
	}
	return tables
}

func ts(now time.Time, seq int) string {
	return now.Add(time.Duration(seq) * time.Second).Format("2006_01_02_150405")
}

func writeMigrationPair(dir, baseName, upSQL, downSQL string) error {
	upPath := filepath.Join(dir, baseName+".up.sql")
	downPath := filepath.Join(dir, baseName+".down.sql")

	if err := os.WriteFile(upPath, []byte(upSQL+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(downPath, []byte(downSQL+"\n"), 0o644); err != nil {
		return err
	}
	return nil
}

func loadSnapshot(path string) (*SchemaSnapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &SchemaSnapshot{Version: 0, Tables: map[string]SnapshotTable{}}, nil
		}
		return nil, err
	}

	var s SchemaSnapshot
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("schema.lock.json invalid: %w", err)
	}
	if s.Tables == nil {
		s.Tables = map[string]SnapshotTable{}
	}
	return &s, nil
}

func saveSnapshot(path string, snap *SchemaSnapshot) error {
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}*/
