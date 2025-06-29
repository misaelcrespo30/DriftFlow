package models

import (
	"time"

	"gorm.io/gorm"
)

// MatterActivityCategory represents a category for matter activities.
type MatterActivityCategory struct {
	gorm.Model
	Name                      string    `gorm:"column:name" json:"name"`
	CreatedByID               uint      `gorm:"column:created_by_id" json:"created_by_id"`
	CreatedOn                 time.Time `gorm:"column:created_on" json:"created_on"`
	Discriminator             string    `gorm:"column:discriminator" json:"discriminator"`
	ModifiedByID              uint      `gorm:"column:modified_by_id" json:"modified_by_id"`
	ModifiedOn                time.Time `gorm:"column:modified_on" json:"modified_on"`
	Rate                      float64   `gorm:"column:rate" json:"rate"`
	BillingMethod             string    `gorm:"column:billing_method" json:"billing_method"`
	CustomRate                float64   `gorm:"column:custom_rate" json:"custom_rate"`
	MatterFlatFeeCategoryRate float64   `gorm:"column:matter_flat_fee_category_rate" json:"matter_flat_fee_category_rate"`
	Field1                    string    `gorm:"column:field1" json:"field1"`
	Field2                    string    `gorm:"column:field2" json:"field2"`
	Field3                    string    `gorm:"column:field3" json:"field3"`
}
