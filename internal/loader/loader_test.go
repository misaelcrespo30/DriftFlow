package loader

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	names := []string{"3.sql", "1.sql", "2.sql", "README.md"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte{}, 0644); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}

	st, err := Load(context.Background(), dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{
		filepath.Join(dir, "1.sql"),
		filepath.Join(dir, "2.sql"),
		filepath.Join(dir, "3.sql"),
	}
	if !reflect.DeepEqual(st.Files, want) {
		t.Fatalf("expected %v, got %v", want, st.Files)
	}
}

func TestLoad_Env(t *testing.T) {
	dir := t.TempDir()
	names := []string{"b.sql", "a.sql"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte{}, 0644); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}

	old := os.Getenv("MIG_DIR")
	t.Setenv("MIG_DIR", dir)
	defer os.Setenv("MIG_DIR", old)

	st, err := Load(context.Background(), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{
		filepath.Join(dir, "a.sql"),
		filepath.Join(dir, "b.sql"),
	}
	if !reflect.DeepEqual(st.Files, want) {
		t.Fatalf("expected %v, got %v", want, st.Files)
	}
}

func TestLoad_MigrationsPath(t *testing.T) {
	dir := t.TempDir()
	names := []string{"x.sql"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte{}, 0644); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}

	old := os.Getenv("MIGRATIONS_PATH")
	t.Setenv("MIGRATIONS_PATH", dir)
	defer os.Setenv("MIGRATIONS_PATH", old)

	st, err := Load(context.Background(), "")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := []string{filepath.Join(dir, "x.sql")}
	if !reflect.DeepEqual(st.Files, want) {
		t.Fatalf("expected %v, got %v", want, st.Files)
	}
}
