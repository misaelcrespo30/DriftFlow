package driftflow

import (
	"reflect"
	"testing"
)

func TestCompareSchemas_TableMissing(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int"},
	}
	tgt := map[string]map[string]string{}

	expect := []string{"missing table: users"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}

func TestCompareSchemas_ColumnMissing(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int", "name": "text"},
	}
	tgt := map[string]map[string]string{
		"users": {"id": "int"},
	}

	expect := []string{"missing column: users.name"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}

func TestCompareSchemas_ColumnExtra(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int"},
	}
	tgt := map[string]map[string]string{
		"users": {"id": "int", "name": "text"},
	}

	expect := []string{"extra column: users.name"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}

func TestCompareSchemas_TypeMismatch(t *testing.T) {
	src := map[string]map[string]string{
		"users": {"id": "int"},
	}
	tgt := map[string]map[string]string{
		"users": {"id": "bigint"},
	}

	expect := []string{"type mismatch for users.id: int vs bigint"}
	got := CompareSchemas(src, tgt)
	if !reflect.DeepEqual(got, expect) {
		t.Fatalf("expected %v, got %v", expect, got)
	}
}
