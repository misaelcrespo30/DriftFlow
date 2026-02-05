package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

const TableNameOIDCClient = "oidc_clients"

// OIDCClient represents an OIDC client configuration.
// - TenantID nil => global client (applies to any tenant)
// - TenantID set => tenant override
type OIDCClient struct {
	// OIDC client_id
	ClientID string `gorm:"column:client_id;size:255;primaryKey" json:"client_id"`
	// Tenant scope (nullable => global)
	TenantID string `gorm:"column:tenant_id;size:36;not null;index:ix_oidc_clients_tenant_id" json:"tenant_id"`
	// App identifier (web/backoffices)
	AppKey string `gorm:"column:app_key;size:100;not null;index:ix_oidc_clients_app_key" json:"app_key"`
	// Helpful unique constraint per tenant+app
	// Note: uniqueness with nullable tenant_id is tricky in Postgres (NULL != NULL).
	// If you want strict uniqueness, implement it via partial unique indexes in migration.
	_ struct{} `gorm:"-:all"` // placeholder, no field

	//["https://web.elevitae.com/auth/callback", "https://web.mydailychoice.com/auth/callback"]
	AllowedRedirectURIs datatypes.JSON `gorm:"column:allowed_redirect_uris;type:jsonb" json:"allowed_redirect_uris"`
	//["openid","email","profile"]
	AllowedScopes                 datatypes.JSON `gorm:"column:allowed_scopes;type:jsonb" json:"allowed_scopes"`
	AllowedPostLogoutRedirectURIs datatypes.JSON `gorm:"column:allowed_post_logout_redirect_uris;type:jsonb" json:"allowed_post_logout_redirect_uris"`
	CreatedAt                     time.Time      `gorm:"column:created_at;not null;index:ix_oidc_settings_created_at" json:"created_at"`
	UpdatedAt                     time.Time      `gorm:"column:updated_at;not null;index:ix_oidc_settings_updated_at" json:"updated_at"`
	DeletedAt                     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (*OIDCClient) TableName() string {
	return TableNameOIDCClient
}
