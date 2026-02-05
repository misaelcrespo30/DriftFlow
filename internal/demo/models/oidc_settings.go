package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

const TableNameOIDCSettings = "oidc_settings"

type OIDCSettings struct {
	ID string `gorm:"column:id;size:36;primaryKey" json:"id"`
	// ["elevitae.com","mydailychoice.com"]
	AllowedRootDomains datatypes.JSON `gorm:"column:allowed_root_domains;type:jsonb" json:"allowed_root_domains"`
	// ["web","backoffices","login"]
	AllowedRedirectSubs datatypes.JSON `gorm:"column:allowed_redirect_subdomains;type:jsonb" json:"allowed_redirect_subdomains"`
	// e.g. "web" or some default client id
	DefaultClientID string         `gorm:"column:default_client_id;size:255;index:ix_oidc_settings_default_client_id" json:"default_client_id"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;index:ix_oidc_settings_created_at" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;not null;index:ix_oidc_settings_updated_at" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (*OIDCSettings) TableName() string {
	return TableNameOIDCSettings
}
