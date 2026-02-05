package models

import (
	"gorm.io/gorm"
	"time"
)

const TableNameService = "services"

type Service struct {
	ServiceID  string         `gorm:"column:service_id;size:36;primaryKey" json:"service_id"`
	ServiceKey string         `gorm:"column:service_key;size:80;not null;uniqueIndex;index:ix_services_key_deleted,priority:1" json:"service_key"`
	Name       string         `gorm:"column:name;size:150;not null;index:ix_services_name" json:"name"`
	IsActive   bool           `gorm:"column:is_active;not null;default:true;index:ix_services_is_active" json:"is_active"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index;index:ix_services_key_deleted,priority:2" json:"-"`
}

func (*Service) TableName() string { return TableNameService }
