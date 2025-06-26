package models

import "gorm.io/gorm"

type MatterRelated struct {
	gorm.Model
	MatterID      uint `gorm:"column:matter_id"`
	ActivityLogID uint `gorm:"column:activity_log_id"`
}
