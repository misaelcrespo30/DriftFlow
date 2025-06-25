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

	// Leer el archivo generado
	data, err := os.ReadFile(filepath.Join(dir, "tmplmodel.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// Deserializar para comparar por estructura
	var got []tmplModel
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}

	// Esperado
	expected := []tmplModel{{Name: "", Age: 0}}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected content:\n got: %#v\nwant: %#v", got, expected)
	}
}

func TestGenerateSeedTemplatesWithData(t *testing.T) {
	dir := t.TempDir()

	// Generadores de datos personalizados
	gens := map[string]func() interface{}{
		"name": func() interface{} { return "alice" },
		"age":  func() interface{} { return 30 },
	}

	err := GenerateSeedTemplatesWithData([]interface{}{tmplModel{}}, dir, gens)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Leer el archivo generado
	data, err := os.ReadFile(filepath.Join(dir, "tmplmodel.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// Deserializar
	var got []tmplModel
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}

	expected := []tmplModel{{Name: "alice", Age: 30}}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected content:\n got: %#v\nwant: %#v", got, expected)
	}
}
