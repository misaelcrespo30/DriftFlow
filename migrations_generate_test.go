package driftflow

import (
	"strings"
	"testing"

	"gorm.io/datatypes"
)

type oidcClientJSONB struct {
	ClientID                      string         `gorm:"column:client_id;primaryKey"`
	AllowedRedirectURIs           datatypes.JSON `gorm:"column:allowed_redirect_uris;type:jsonb"`
	AllowedScopes                 datatypes.JSON `gorm:"column:allowed_scopes"`
	AllowedPostLogoutRedirectURIs datatypes.JSON `gorm:"column:allowed_post_logout_redirect_uris"`
}

func TestBuildModelSchemaJSONBColumns(t *testing.T) {
	_, orderMap, defMap, _, _, err := buildModelSchema([]interface{}{oidcClientJSONB{}}, "postgres")
	if err != nil {
		t.Fatalf("buildModelSchema: %v", err)
	}

	defs, ok := defMap["oidc_clients"]
	if !ok {
		t.Fatalf("expected oidc_clients table in schema")
	}

	for _, col := range []string{
		"allowed_redirect_uris",
		"allowed_scopes",
		"allowed_post_logout_redirect_uris",
	} {
		if defs[col] != "jsonb" {
			t.Fatalf("expected %s jsonb, got %q", col, defs[col])
		}
	}

	sql := createTableSQL("oidc_clients", defs, orderMap["oidc_clients"], nil, "postgres")
	if !strings.Contains(sql, `"allowed_redirect_uris" jsonb`) {
		t.Fatalf("expected create table to include allowed_redirect_uris jsonb, got: %s", sql)
	}
	if !strings.Contains(sql, `"allowed_scopes" jsonb`) {
		t.Fatalf("expected create table to include allowed_scopes jsonb, got: %s", sql)
	}
	if !strings.Contains(sql, `"allowed_post_logout_redirect_uris" jsonb`) {
		t.Fatalf("expected create table to include allowed_post_logout_redirect_uris jsonb, got: %s", sql)
	}
}
