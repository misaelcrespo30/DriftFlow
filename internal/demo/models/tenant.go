package models

import (
	"gorm.io/gorm"
	"time"
)

const TableNameTenant = "tenants"

type Tenant struct {
	TenantID               string          `gorm:"column:tenant_id;size:36;primaryKey" json:"tenant_id"`
	TenantName             string          `gorm:"column:tenant_name;size:200;not null;index:ix_tenants_tenant_name,priority:1" json:"tenant_name"`
	ServicePlan            string          `gorm:"column:service_plan;size:50;not null;default:'standard';index:ix_tenants_service_plan" json:"service_plan"`
	RecoveryState          string          `gorm:"column:recovery_state;size:50;not null;index:ix_tenants_recovery_state" json:"recovery_state"`
	LastUpdated            time.Time       `gorm:"column:last_updated;not null;index:ix_tenants_last_updated" json:"last_updated"`
	AccessCodeHash         string          `gorm:"column:access_code_hash;size:250" json:"-"`
	IsAvailable            bool            `gorm:"column:is_available;not null;default:false;index:ix_tenants_is_available" json:"is_available"`
	IsDataSeeded           bool            `gorm:"column:is_data_seeded;not null;default:false;index:ix_tenants_is_data_seeded" json:"is_data_seeded"`
	ShouldDelete           bool            `gorm:"column:should_delete;not null;default:false;index:ix_tenants_should_delete" json:"should_delete"`
	QuantityOfEmployees    int64           `gorm:"column:quantity_of_employees;not null;default:0" json:"quantity_of_employees"`
	PlanID                 *string         `gorm:"column:plan_id;size:36;index:ix_tenants_plan_id,priority:1" json:"plan_id"`
	SeatsAllowed           int64           `gorm:"column:seats_allowed;not null;default:0" json:"seats_allowed"`
	SeatsUsed              int64           `gorm:"column:seats_used;not null;default:0" json:"seats_used"`
	SubscriptionCustomerID *string         `gorm:"column:subscription_customer_id;size:100;index:ix_tenants_subscription_customer_id" json:"subscription_customer_id"`
	SubscriptionID         *string         `gorm:"column:subscription_id;size:100;index:ix_tenants_subscription_id" json:"subscription_id"`
	SubscriptionItemID     *string         `gorm:"column:subscription_item_id;size:100" json:"subscription_item_id"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
	DeletedAt              gorm.DeletedAt  `gorm:"index" json:"-"`
	Plan                   *Plan           `gorm:"foreignKey:PlanID;references:PlanID" json:"plan,omitempty"`
	TenantUsers            []TenantUser    `gorm:"foreignKey:TenantID;references:TenantID" json:"tenant_users,omitempty"`
	TenantServices         []TenantService `gorm:"foreignKey:TenantID;references:TenantID" json:"tenant_services,omitempty"`
}

// TableName Tenant's table name
func (*Tenant) TableName() string {
	return TableNameTenant
}
