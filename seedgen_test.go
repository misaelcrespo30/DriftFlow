package driftflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type tmplModel struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestGenerateSeedTemplates(t *testing.T) {
	dir := t.TempDir()

	err := GenerateSeedTemplates([]interface{}{tmplModel{}}, dir)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "tmplmodel.seed.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var got []tmplModel
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}

	if len(got) != 10 {
		t.Fatalf("expected 10 items, got %d", len(got))
	}
	for i, item := range got {
		if item.Age != i+1 {
			t.Fatalf("unexpected age at %d: %d", i, item.Age)
		}
	}
}

func TestGenerateSeedTemplatesWithData(t *testing.T) {
	dir := t.TempDir()

	gens := map[string]func() interface{}{
		"name": func() interface{} { return "alice" },
		"age":  func() interface{} { return 30 },
	}

	err := GenerateSeedTemplatesWithData([]interface{}{tmplModel{}}, dir, gens)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "tmplmodel.seed.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var got []tmplModel
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}

	if len(got) != 10 {
		t.Fatalf("expected 10 items, got %d", len(got))
	}
	if got[0].Name != "alice" || got[0].Age != 30 {
		t.Fatalf("unexpected first item: %#v", got[0])
	}
}
