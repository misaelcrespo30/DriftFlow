package driftflow

import (
	"reflect"
	"testing"
)

type project struct {
	ID       string          `gorm:"column:id;primaryKey"`
	Settings *projectSetting `gorm:"foreignKey:ProjectID;references:ID"`
	Members  []projectMember `gorm:"foreignKey:ProjectID;references:ID"`
}

type projectSetting struct {
	ID        string   `gorm:"column:id;primaryKey"`
	ProjectID string   `gorm:"column:project_id"`
	Project   *project `gorm:"foreignKey:ProjectID;references:ID"`
}

type projectMember struct {
	ID        string `gorm:"column:id;primaryKey"`
	ProjectID string `gorm:"column:project_id"`
}

func TestToSnakeWithInitialisms(t *testing.T) {
	cases := map[string]string{
		"ProjectID":    "project_id",
		"URLValue":     "url_value",
		"HTTPServerID": "http_server_id",
	}

	for in, want := range cases {
		if got := toSnakeWithInitialisms(in); got != want {
			t.Fatalf("toSnakeWithInitialisms(%q) = %q, want %q", in, got, want)
		}
		if got := toSnakeCase(in); got != want {
			t.Fatalf("toSnakeCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInferRelationKinds(t *testing.T) {
	projectType := reflect.TypeOf(project{})
	settingType := reflect.TypeOf(projectSetting{})

	settingsField, _ := projectType.FieldByName("Settings")
	rel := inferRelation(projectType, settingsField)
	if rel.Kind != relationHasOne || rel.OwnerTable != "project_settings" || rel.ForeignKeyColumn != "project_id" || rel.ReferencesTable != "projects" || rel.ReferencesColumn != "id" {
		t.Fatalf("has-one relation mismatch: %+v", rel)
	}

	membersField, _ := projectType.FieldByName("Members")
	rel = inferRelation(projectType, membersField)
	if rel.Kind != relationHasMany || rel.OwnerTable != "project_members" || rel.ForeignKeyColumn != "project_id" || rel.ReferencesTable != "projects" || rel.ReferencesColumn != "id" {
		t.Fatalf("has-many relation mismatch: %+v", rel)
	}

	projectField, _ := settingType.FieldByName("Project")
	rel = inferRelation(settingType, projectField)
	if rel.Kind != relationBelongsTo || rel.OwnerTable != "project_settings" || rel.ForeignKeyColumn != "project_id" || rel.ReferencesTable != "projects" || rel.ReferencesColumn != "id" {
		t.Fatalf("belongs-to relation mismatch: %+v", rel)
	}
}

func TestBuildModelSchemaPlacesForeignKeysInChildTable(t *testing.T) {
	_, _, _, fkMap, _, err := buildModelSchema([]interface{}{project{}, projectSetting{}, projectMember{}}, "postgres")
	if err != nil {
		t.Fatalf("buildModelSchema: %v", err)
	}

	projectFKs := fkMap["projects"]
	if len(projectFKs) != 0 {
		t.Fatalf("expected no FKs in projects, got %+v", projectFKs)
	}

	settingsFKs := fkMap["project_settings"]
	if len(settingsFKs) == 0 {
		t.Fatalf("expected FKs in project_settings")
	}
	foundSettingsFK := false
	for _, fk := range settingsFKs {
		if fk.Column == "project_id" && fk.RefTable == "projects" && fk.RefColumn == "id" {
			foundSettingsFK = true
			break
		}
	}
	if !foundSettingsFK {
		t.Fatalf("expected project_settings.project_id -> projects.id FK, got %+v", settingsFKs)
	}

	membersFKs := fkMap["project_members"]
	if len(membersFKs) == 0 {
		t.Fatalf("expected FKs in project_members")
	}
	foundMembersFK := false
	for _, fk := range membersFKs {
		if fk.Column == "project_id" && fk.RefTable == "projects" && fk.RefColumn == "id" {
			foundMembersFK = true
			break
		}
	}
	if !foundMembersFK {
		t.Fatalf("expected project_members.project_id -> projects.id FK, got %+v", membersFKs)
	}
}

func TestDedupeForeignKeys(t *testing.T) {
	fks := []foreignKeyInfo{
		{Column: "project_id", RefTable: "projects", RefColumn: "id"},
		{Column: "project_id", RefTable: "projects", RefColumn: "id"},
		{Column: "owner_id", RefTable: "users", RefColumn: "id"},
	}
	got := dedupeForeignKeys(fks)
	if len(got) != 2 {
		t.Fatalf("expected 2 deduped FKs, got %d: %+v", len(got), got)
	}
}

func TestOrderTablesByFKDependencies(t *testing.T) {
	tables := []string{"project_settings", "projects"}
	fkMap := map[string][]foreignKeyInfo{
		"project_settings": {
			{Column: "project_id", RefTable: "projects", RefColumn: "id"},
		},
	}
	ordered := orderTablesByFKDependencies(tables, fkMap)
	if len(ordered) != 2 || ordered[0] != "projects" || ordered[1] != "project_settings" {
		t.Fatalf("unexpected order: %+v", ordered)
	}
}
