package driftflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestGenerateSeedTemplatesOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tmplmodel.seed.json")

	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("prewrite: %v", err)
	}

	if err := GenerateSeedTemplates([]interface{}{tmplModel{}}, dir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	data, err := os.ReadFile(path)
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
}

func TestRuleKeyFieldAvoidsUUID(t *testing.T) {
	value := dummyValueForField("service_key", reflect.TypeOf(""), 0, time.Now())
	strValue, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}
	if _, err := uuid.Parse(strValue); err == nil {
		t.Fatalf("expected non-uuid value, got %s", strValue)
	}
}

func TestRuleDomainFieldUsesKnownDomains(t *testing.T) {
	value := dummyValueForField("custom_domain", reflect.TypeOf(""), 1, time.Now())
	strValue, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}
	allowed := allowedRootDomains()
	found := false
	for _, domain := range allowed {
		if strValue == domain {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected domain in %v, got %s", allowed, strValue)
	}
}

func TestRuleAllowedRedirectURIsJSON(t *testing.T) {
	value := dummyValueForField("allowed_redirect_uris", reflect.TypeOf(JSON{}), 0, time.Now())
	raw, ok := value.(json.RawMessage)
	if !ok {
		t.Fatalf("expected json.RawMessage, got %T", value)
	}
	expected, err := json.Marshal(allowedRedirectURIs())
	if err != nil {
		t.Fatalf("marshal expected: %v", err)
	}
	if string(raw) != string(expected) {
		t.Fatalf("expected %s, got %s", expected, raw)
	}
}

func TestDedupeSeedObjectsByNaturalKey(t *testing.T) {
	obj1 := newOrderedMap()
	obj1.set("service_key", "branding")
	obj2 := newOrderedMap()
	obj2.set("service_key", "branding")
	obj3 := newOrderedMap()
	obj3.set("service_key", "network")

	objs := dedupeSeedObjects([]*orderedMap{obj1, obj2, obj3}, []string{"service_key"})
	if len(objs) != 2 {
		t.Fatalf("expected 2 unique objects, got %d", len(objs))
	}
}

type JSON []byte
