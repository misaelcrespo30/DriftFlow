package models

import (
	"time"

	"gorm.io/gorm"
)

type MatterActivity struct {
	gorm.Model
	UserID            uint      `gorm:"column:user_id" json:"user_id"`
	MatterID          uint      `gorm:"column:matter_id" json:"matter_id"`
	Date              time.Time `gorm:"column:date" json:"date"`
	Description       string    `gorm:"column:description" json:"description"`
	Rate              float64   `gorm:"column:rate" json:"rate"`
	CreatedByID       uint      `gorm:"column:created_by_id" json:"created_by_id"`
	CreatedOn         time.Time `gorm:"column:created_on" json:"created_on"`
	ModifiedByID      uint      `gorm:"column:modified_by_id" json:"modified_by_id"`
	ModifiedOn        time.Time `gorm:"column:modified_on" json:"modified_on"`
	EventEntryID      uint      `gorm:"column:event_entry_id" json:"event_entry_id"`
	MatterNoteID      uint      `gorm:"column:matter_note_id" json:"matter_note_id"`
	TaskID            uint      `gorm:"column:task_id" json:"task_id"`
	CategoryID        uint      `gorm:"column:category_id" json:"category_id"`
	ActivityType      string    `gorm:"column:activity_type" json:"activity_type"`
	Amount            float64   `gorm:"column:amount" json:"amount"`
	Code              string    `gorm:"column:code" json:"code"`
	MatterID1         uint      `gorm:"column:matter_id1" json:"matter_id1"`
	BillID            uint      `gorm:"column:bill_id" json:"bill_id"`
	Duration          uint      `gorm:"column:duration" json:"duration"`
	StartedAt         time.Time `gorm:"column:started_at" json:"started_at"`
	MatterFlatFeeCode string    `gorm:"column:matter_flat_fee_code" json:"matter_flat_fee_code"`
	IsMain            bool      `gorm:"column:is_main" json:"is_main"`
	Field1            string    `gorm:"column:field1" json:"field1"`
	Field2            string    `gorm:"column:field2" json:"field2"`
	Field3            string    `gorm:"column:field3" json:"field3"`
	IsBillable        bool      `gorm:"column:is_billable" json:"is_billable"`
	Charge            float64   `gorm:"column:charge" json:"charge"`
	NoMatter          bool      `gorm:"column:no_matter" json:"no_matter"`
}
