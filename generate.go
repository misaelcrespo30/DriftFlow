package driftflow

import (
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
	Name       string `json:"name"` // filename, incluye .sql
	SQLSHA256  string `json:"sql_sha256"`
	CreatedUTC string `json:"created_utc"`
}

type ManifestIssueType string

type ManifestIssue struct {
	Type      ManifestIssueType
	Migration string // filename
	File      string // path si aplica
	Detail    string
}

type manifestLockWire struct {
	Version    int                 `json:"version"`
	Migrations []manifestEntryWire `json:"migrations"`
	UpdatedUTC string              `json:"updated_utc,omitempty"`
}

type manifestEntryWire struct {
	Name       string `json:"name"`
	SQLSHA256  string `json:"sql_sha256"`
	UpSHA256   string `json:"up_sha256"`
	DownSHA256 string `json:"down_sha256"`
	CreatedUTC string `json:"created_utc"`
}

func loadManifest(path string) (*ManifestLock, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ManifestLock{Version: 0, Migrations: []ManifestEntry{}}, nil
		}
		return nil, err
	}
	var wire manifestLockWire
	if err := json.Unmarshal(b, &wire); err != nil {
		return nil, fmt.Errorf("manifest.lock.json invalid: %w", err)
	}
	m := ManifestLock{
		Version:    wire.Version,
		UpdatedUTC: wire.UpdatedUTC,
		Migrations: make([]ManifestEntry, 0, len(wire.Migrations)),
	}
	for _, entry := range wire.Migrations {
		name := normalizeManifestName(entry.Name)
		m.Migrations = append(m.Migrations, ManifestEntry{
			Name:       name,
			SQLSHA256:  entry.SQLSHA256,
			CreatedUTC: entry.CreatedUTC,
		})
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

func normalizeManifestName(name string) string {
	if name == "" {
		return ""
	}
	if strings.HasSuffix(name, ".sql") {
		return name
	}
	return name + ".sql"
}

func hashMigrationFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sha256Hex(b), nil
}

func migrateManifest(dir string, manifest *ManifestLock) (bool, error) {
	changed := false
	for i := range manifest.Migrations {
		entry := &manifest.Migrations[i]
		normalized := normalizeManifestName(entry.Name)
		if normalized != entry.Name {
			entry.Name = normalized
			changed = true
		}
		if entry.SQLSHA256 == "" && entry.Name != "" {
			hash, err := hashMigrationFile(filepath.Join(dir, entry.Name))
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return false, err
			}
			entry.SQLSHA256 = hash
			changed = true
		}
	}
	if changed {
		manifest.Version++
	}
	return changed, nil
}

func validateManifest(dir string, manifest *ManifestLock) ([]ManifestIssue, error) {
	entries := make(map[string]ManifestEntry, len(manifest.Migrations))
	for _, e := range manifest.Migrations {
		if e.Name == "" {
			return nil, fmt.Errorf("manifest has empty migration name")
		}
		if !strings.HasSuffix(e.Name, ".sql") {
			return nil, fmt.Errorf("manifest migration name missing .sql: %s", e.Name)
		}
		if _, exists := entries[e.Name]; exists {
			return nil, fmt.Errorf("manifest has duplicate migration name: %s", e.Name)
		}
		entries[e.Name] = e
	}

	var issues []ManifestIssue

	// 1) scan disk and ensure no untracked/missing pairs exist
	files, _ := filepath.Glob(filepath.Join(dir, "*.sql"))

	diskNames := map[string]struct{}{}
	for _, p := range files {
		name := filepath.Base(p)
		diskNames[name] = struct{}{}
	}

	for name := range entries {
		if _, ok := diskNames[name]; !ok {
			issues = append(issues, ManifestIssue{Type: IssueMissingPair, Migration: name, Detail: "manifest entry missing migration file"})
		}
	}

	for name := range diskNames {
		if _, ok := entries[name]; !ok {
			issues = append(issues, ManifestIssue{Type: IssueUntracked, Migration: name, Detail: "migration exists on disk but is not registered in manifest"})
		}
	}

	// 2) verify tracked entries hashes
	for name, e := range entries {
		if _, ok := diskNames[name]; !ok {
			continue
		}
		path := filepath.Join(dir, name)

		hash, err := hashMigrationFile(path)
		if err != nil {
			return nil, err
		}

		if !strings.EqualFold(hash, e.SQLSHA256) {
			issues = append(issues, ManifestIssue{Type: IssueHashMismatch, Migration: name, File: path, Detail: "SQL hash mismatch"})
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

	recalc := func(name string) (string, error) {
		path := filepath.Join(dir, name)
		hash, err := hashMigrationFile(path)
		if err != nil {
			return "", err
		}
		return hash, nil
	}

	// fix mismatches
	for _, is := range issues {
		if is.Type != IssueHashMismatch {
			continue
		}
		name := is.Migration
		entry := byName[name]
		if entry == nil {
			continue
		}
		hash, err := recalc(name)
		if err != nil {
			return err
		}
		entry.SQLSHA256 = hash
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
			name := normalizeManifestName(is.Migration)
			if seen[name] {
				continue
			}
			hash, err := recalc(name)
			if err != nil {
				return err
			}
			manifest.Migrations = append(manifest.Migrations, ManifestEntry{
				Name:       name,
				SQLSHA256:  hash,
				CreatedUTC: time.Now().UTC().Format(time.RFC3339),
			})
			seen[name] = true
		}
	}

	sort.SliceStable(manifest.Migrations, func(i, j int) bool {
		return manifest.Migrations[i].Name < manifest.Migrations[j].Name
	})

	manifest.Version++
	return nil
}

func appendMigrationToManifest(dir string, manifest *ManifestLock, baseName, createdUTC string) error {
	name := normalizeManifestName(baseName)
	path := filepath.Join(dir, name)
	hash, err := hashMigrationFile(path)
	if err != nil {
		return err
	}

	manifest.Migrations = append(manifest.Migrations, ManifestEntry{
		Name:       name,
		SQLSHA256:  hash,
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
	migrated, err := migrateManifest(dir, manifest)
	if err != nil {
		return err
	}
	if migrated {
		if err := saveManifest(manifestPath, manifest); err != nil {
			return err
		}
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

			if err := writeMigrationFile(dir, name, up, down); err != nil {
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

		if err := writeMigrationFile(dir, name, up, down); err != nil {
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
