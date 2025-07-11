package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDir(t *testing.T) {
	dir := t.TempDir()
	if err := ValidateDir(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := ValidateDir(dir); err == nil {
		t.Fatalf("expected error for missing dir")
	}
}

func TestValidateDirs(t *testing.T) {
	base := t.TempDir()
	m := filepath.Join(base, "m")
	s := filepath.Join(base, "s")
	os.Mkdir(m, 0o755)
	os.Mkdir(s, 0o755)
	if err := ValidateDirs(m, s); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	os.RemoveAll(m)
	if err := ValidateDirs(m, s); err == nil {
		t.Fatalf("expected error for missing mig dir")
	}
}

func TestLoad_MigrationsPath(t *testing.T) {
	old := os.Getenv("MIGRATIONS_PATH")
	t.Setenv("MIGRATIONS_PATH", "/tmp/migs")
	defer os.Setenv("MIGRATIONS_PATH", old)

	cfg := Load()
	if cfg.MigDir != "/tmp/migs" {
		t.Fatalf("expected /tmp/migs, got %s", cfg.MigDir)
	}
}
