package driftflow

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

type mockSeeder struct {
	path string
}

func (m *mockSeeder) Seed(_ *gorm.DB, p string) error {
	m.path = p
	return nil
}

func TestSeed(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	m := &mockSeeder{}
	if err := Seed(db, dir, []Seeder{m}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	expected := filepath.Join(dir, "mockseeder.json")
	if m.path != expected {
		t.Fatalf("expected %s, got %s", expected, m.path)
	}
}
