package models

import (
	"time"

	"gorm.io/gorm"
)

type MatterActivity struct {
	gorm.Model
	UserID            uint      `gorm:"column:user_id"`
	MatterID          uint      `gorm:"column:matter_id"`
	Date              time.Time `gorm:"column:date"`
	Description       string    `gorm:"column:description"`
	Rate              float64   `gorm:"column:rate"`
	CreatedByID       uint      `gorm:"column:created_by_id"`
	CreatedOn         time.Time `gorm:"column:created_on"`
	ModifiedByID      uint      `gorm:"column:modified_by_id"`
	ModifiedOn        time.Time `gorm:"column:modified_on"`
	EventEntryID      uint      `gorm:"column:event_entry_id"`
	MatterNoteID      uint      `gorm:"column:matter_note_id"`
	TaskID            uint      `gorm:"column:task_id"`
	CategoryID        uint      `gorm:"column:category_id"`
	ActivityType      string    `gorm:"column:activity_type"`
	Amount            float64   `gorm:"column:amount"`
	Code              string    `gorm:"column:code"`
	MatterID1         uint      `gorm:"column:matter_id1"`
	BillID            uint      `gorm:"column:bill_id"`
	Duration          uint      `gorm:"column:duration"`
	StartedAt         time.Time `gorm:"column:started_at"`
	MatterFlatFeeCode string    `gorm:"column:matter_flat_fee_code"`
	IsMain            bool      `gorm:"column:is_main"`
	Field1            string    `gorm:"column:field1"`
	Field2            string    `gorm:"column:field2"`
	Field3            string    `gorm:"column:field3"`
	IsBillable        bool      `gorm:"column:is_billable"`
	Charge            float64   `gorm:"column:charge"`
	NoMatter          bool      `gorm:"column:no_matter"`
}
