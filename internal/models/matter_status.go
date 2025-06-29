package models

import "gorm.io/gorm"

// MatterStatus represents a status that can be assigned to a matter.
type MatterStatus struct {
	gorm.Model
	Name           string `gorm:"column:name" json:"name"`
	IsSystem       bool   `gorm:"column:is_system" json:"is_system"`
	IsNoteRequired bool   `gorm:"column:is_note_required" json:"is_note_required"`
	Color          string `gorm:"column:color" json:"color"`
}
