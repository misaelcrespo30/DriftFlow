package models

import (
	"gorm.io/gorm"
	"time"
)

const TableNameTenantUser = "tenant_users"

type TenantUser struct {
	ID                 string         `gorm:"column:id;size:36;primaryKey" json:"id"`
	UserID             string         `gorm:"column:user_id;size:36;not null;index:ix_tenant_users_user_id,priority:1;index:ix_tenant_users_user_tenant,priority:1" json:"user_id"`
	TenantID           string         `gorm:"column:tenant_id;size:36;not null;index:ix_tenant_users_tenant_id,priority:1;index:ix_tenant_users_user_tenant,priority:2" json:"tenant_id"`
	Relationship       int64          `gorm:"column:relationship;not null" json:"relationship"`
	IsActive           bool           `gorm:"column:is_active;not null;default:false;index:ix_tenant_users_user_active,priority:2" json:"is_active"`
	IsDefault          bool           `gorm:"column:is_default;not null;default:false;index:ix_tenant_users_user_default,priority:2" json:"is_default"`
	OriginatedUser     bool           `gorm:"column:originated_user;not null;default:false" json:"originated_user"`
	ExternalIdentityID string         `gorm:"column:external_identity_id;size:100;not null;index:ix_tenant_users_external_identity" json:"external_identity_id"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index;index:ix_tenant_users_user_active,priority:3;index:ix_tenant_users_user_default,priority:3" json:"-"`
	User               *User          `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
	Tenant             *Tenant        `gorm:"foreignKey:TenantID;references:TenantID" json:"tenant,omitempty"`
}

func (*TenantUser) TableName() string {
	return TableNameTenantUser
}
