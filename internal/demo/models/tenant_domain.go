package models

import (
	"gorm.io/gorm"
	"time"
)

const TableNameTenantDomain = "tenant_domains"

type TenantDomain struct {
	ID         string         `gorm:"column:id;size:36;primaryKey" json:"id"`
	Domain     string         `gorm:"column:domain;size:255;not null;uniqueIndex;index:ix_tenant_domains_domain_tenant,priority:1" json:"domain"`
	TenantID   string         `gorm:"column:tenant_id;size:36;not null;index:ix_tenant_domains_tenant_id,priority:1;index:ix_tenant_domains_domain_tenant,priority:2" json:"tenant_id"`
	IsPrimary  bool           `gorm:"column:is_primary;not null;default:false;index:ix_tenant_domains_is_primary" json:"is_primary"`
	IsVerified bool           `gorm:"column:is_verified;not null;default:false;index:ix_tenant_domains_is_verified" json:"is_verified"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Tenant     *Tenant        `gorm:"foreignKey:TenantID;references:TenantID" json:"tenant,omitempty"`
}

func (*TenantDomain) TableName() string {
	return TableNameTenantDomain
}
