package driftflow

import (
	"reflect"
	"strings"
	"testing"
)

func TestColumnDef_UniqueIndexDoesNotStampColumnUnique(t *testing.T) {
	type row struct {
		TenantID   string `gorm:"column:tenant_id;size:36;not null;uniqueIndex:uidx_map_usage,priority:1"`
		Provider   string `gorm:"column:provider;size:80;not null;uniqueIndex:uidx_map_usage,priority:2"`
		BareUnique string `gorm:"column:code;size:40;not null;unique"`
	}
	rt := reflect.TypeOf(row{})
	tenant, _ := rt.FieldByName("TenantID")
	provider, _ := rt.FieldByName("Provider")
	bare, _ := rt.FieldByName("BareUnique")

	_, tenantFull := columnDef(tenant, "postgres", false)
	_, providerFull := columnDef(provider, "postgres", false)
	_, bareFull := columnDef(bare, "postgres", false)

	if defHasUniqueToken(tenantFull) {
		t.Fatalf("composite uniqueIndex must not stamp column unique: %q", tenantFull)
	}
	if defHasUniqueToken(providerFull) {
		t.Fatalf("composite uniqueIndex must not stamp column unique: %q", providerFull)
	}
	if !defHasUniqueToken(bareFull) {
		t.Fatalf("bare unique tag must stamp column unique: %q", bareFull)
	}
}

func TestBuildAlterSQL_DropsObsoleteColumnUnique(t *testing.T) {
	altered := map[string]ColAlter{
		"tenant_id": {
			From: "varchar(36) not null unique",
			To:   "varchar(36) not null",
		},
		"provider": {
			From: "varchar(80) not null unique",
			To:   "varchar(80) not null",
		},
	}
	up, down := buildAlterSQLWithEngine(
		`"map_usage_counters"`,
		"map_usage_counters",
		"postgres",
		nil, nil, nil, nil, nil, altered,
	)
	if !strings.Contains(up, `DROP CONSTRAINT IF EXISTS "map_usage_counters_tenant_id_key"`) {
		t.Fatalf("up missing tenant drop: %s", up)
	}
	if !strings.Contains(up, `DROP CONSTRAINT IF EXISTS "map_usage_counters_provider_key"`) {
		t.Fatalf("up missing provider drop: %s", up)
	}
	if strings.Contains(up, "ALTER COLUMN") {
		t.Fatalf("uniqueness-only change must not ALTER COLUMN TYPE: %s", up)
	}
	if !strings.Contains(down, `ADD CONSTRAINT "map_usage_counters_tenant_id_key" UNIQUE ("tenant_id")`) {
		t.Fatalf("down missing tenant add: %s", down)
	}
}
