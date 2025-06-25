package driftflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type tmplModel struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestGenerateSeedTemplates(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateSeedTemplates([]interface{}{tmplModel{}}, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "tmplmodel.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := strings.TrimSpace(string(data))
	expect := strings.TrimSpace(`[
  {
    "name": "",
    "age": 0
  }
]`)
	if got != expect {
		t.Fatalf("unexpected file:\n%s", got)
	}
}
