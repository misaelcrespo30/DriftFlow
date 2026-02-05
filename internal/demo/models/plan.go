package models

import (
	"gorm.io/gorm"
	"time"
)

const TableNamePlan = "plans"

type Plan struct {
	PlanID     string         `gorm:"column:plan_id;size:36;primaryKey" json:"plan_id"`
	Name       string         `gorm:"column:name;size:100;not null;uniqueIndex;index:ix_plans_name_deleted,priority:1" json:"name" validate:"required"`
	ExternalID *string        `gorm:"column:external_id;size:100;index:ix_plans_external_id" json:"external_id,omitempty"`
	MaxSeats   int            `gorm:"column:max_seats;not null" json:"max_seats" validate:"required,gte=1"`
	MinSeats   int            `gorm:"column:min_seats;not null" json:"min_seats" validate:"required,gte=1,ltefield=MaxSeats"`
	IsDisabled bool           `gorm:"column:is_disabled;not null;default:false;index:ix_plans_is_disabled" json:"is_disabled"`
	Version    string         `gorm:"column:version;size:20;index:ix_plans_version" json:"version"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;not null" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index;index:ix_plans_name_deleted,priority:2" json:"-"`
	Tenants    []Tenant       `gorm:"foreignKey:PlanID;references:PlanID" json:"tenants,omitempty"`
}

func (*Plan) TableName() string {
	return TableNamePlan
}
