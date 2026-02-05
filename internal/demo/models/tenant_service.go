package models

import (
	"gorm.io/gorm"
	"time"
)

const TableNameTenantServiceDatastore = "tenant_service"

type TenantService struct {
	ID               string         `gorm:"column:id;size:36;primaryKey" json:"id"`
	TenantID         string         `gorm:"column:tenant_id;size:36;not null;index:ix_tsd_tenant_service,priority:1;index:ix_tsd_tenant,priority:1" json:"tenant_id"`
	ServiceID        string         `gorm:"column:service_id;size:36;not null;index:ix_tsd_tenant_service,priority:2;index:ix_tsd_service,priority:1" json:"service_id"`
	ConnectionString string         `gorm:"column:connection_string;type:text;not null" json:"connection_string"`
	DBType           string         `gorm:"column:db_type;size:30;not null;default:'postgres';index:ix_tsd_db_type" json:"db_type"`
	Region           *string        `gorm:"column:region;size:40;index:ix_tsd_region" json:"region,omitempty"`
	IsPrimary        bool           `gorm:"column:is_primary;not null;default:true;index:ix_tsd_primary" json:"is_primary"`
	RotationVersion  int64          `gorm:"column:rotation_version;not null;default:1;index:ix_tsd_rotation_version" json:"rotation_version"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Tenant           *Tenant        `gorm:"foreignKey:TenantID;references:TenantID" json:"tenant,omitempty"`
	Service          *Service       `gorm:"foreignKey:ServiceID;references:ServiceID" json:"service,omitempty"`
}

func (*TenantService) TableName() string { return TableNameTenantServiceDatastore }
