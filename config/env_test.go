package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureEnvFileCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := EnsureEnvFile(path); err != nil {
		t.Fatalf("EnsureEnvFile: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != defaultEnvContent {
		t.Fatalf("unexpected content:\n%s", string(b))
	}
}

func TestEnsureEnvFileNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	initial := []byte("EXISTING=1\n")
	if err := os.WriteFile(path, initial, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := EnsureEnvFile(path); err != nil {
		t.Fatalf("EnsureEnvFile: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != string(initial) {
		t.Fatalf("file overwritten")
	}
}
