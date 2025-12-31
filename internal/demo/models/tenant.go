package models

import (
	"time"
)

const TableNameTenant = "Tenants"

type Tenant struct {
	TenantID string `gorm:"column:tenant_id;size:36;primaryKey" json:"tenant_id"`

	TenantName string `gorm:"column:tenant_name;size:200;not null;index:ix_tenants_tenant_name,priority:1" json:"tenant_name"`

	ServicePlan   string    `gorm:"column:service_plan;size:50;not null;default:'standard'" json:"service_plan"`
	RecoveryState string    `gorm:"column:recovery_state;size:50;not null" json:"recovery_state"`
	LastUpdated   time.Time `gorm:"column:last_updated;not null" json:"last_updated"`

	ConnectionString string `gorm:"column:connection_string;type:text;not null" json:"connection_string"`

	IsAvailable  bool `gorm:"column:is_available;not null;default:false" json:"is_available"`
	IsDataSeeded bool `gorm:"column:is_data_seeded;not null;default:false" json:"is_data_seeded"`
	ShouldDelete bool `gorm:"column:should_delete;not null;default:false" json:"should_delete"`

	FirmName            string `gorm:"column:firm_name;size:200" json:"firm_name"`
	QuantityOfEmployees int64  `gorm:"column:quantity_of_employees;not null;default:0" json:"quantity_of_employees"`

	PlanID *string `gorm:"column:plan_id;size:36;index:ix_tenants_plan_id,priority:1" json:"plan_id"`

	SeatsAllowed int64 `gorm:"column:seats_allowed;not null;default:0" json:"seats_allowed"`
	SeatsUsed    int64 `gorm:"column:seats_used;not null;default:0" json:"seats_used"`

	SubscriptionCustomerID *string `gorm:"column:subscription_customer_id;size:100" json:"subscription_customer_id"`
	SubscriptionID         *string `gorm:"column:subscription_id;size:100" json:"subscription_id"`
	SubscriptionItemID     *string `gorm:"column:subscription_item_id;size:100" json:"subscription_item_id"`

	Misael string `gorm:"column:misael;size:50;not null;default:'standard'" json:"misael"`
	// Relaciones
	Plan        *Plan        `gorm:"foreignKey:PlanID;references:PlanID" json:"plan,omitempty"`
	TenantUsers []TenantUser `gorm:"foreignKey:TenantID;references:TenantID" json:"tenant_users,omitempty"`

	Padron string `gorm:"column:padron;size:100;uniqueIndex" json:"padron"`
}

// TableName Tenant's table name
func (*Tenant) TableName() string {
	return TableNameTenant
}

/*type Tenant struct {
	ID                     uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantId               string         `gorm:"type:string;default:uuid_generate_v4();uniqueIndex" json:"tenant_id"`
	TenantName             string         `gorm:"type:varchar(100);not null" json:"tenant_name" validate:"required"`
	ConnectionKey          string         `gorm:"unique;type:varchar(64)" json:"connection_key" validate:"required"`
	ConnectionString       string         `gorm:"type:text;not null" json:"connection_string" validate:"required"`
	Available              bool           `gorm:"default:true" json:"available"`
	SeatsAllowed           int            `gorm:"default:1" json:"seats_allowed"`
	SeatsUsed              int            `gorm:"default:0" json:"seats_used"`
	SubscriptionId         *string        `gorm:"type:varchar(100)" json:"subscription_id,omitempty"`
	SubscriptionCustomerId *string        `gorm:"type:varchar(100)" json:"subscription_customer_id,omitempty"`
	SubscriptionItemId     *string        `gorm:"type:varchar(100)" json:"subscription_item_id,omitempty"`
	PlanId                 *int           `gorm:"index" json:"plan_id"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`

	TenantUsers []TenantUser `gorm:"foreignKey:TenantID;references:TenantId" json:"tenant_users,omitempty"`
	Plan        *Plan        `gorm:"foreignKey:PlanId;references:ID" json:"plan,omitempty"`
}*/
