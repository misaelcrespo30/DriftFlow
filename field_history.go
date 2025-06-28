package driftflow

import (
	"gorm.io/gorm"
	"time"
)

// FieldHistory represents a change to a table column.
type FieldHistory struct {
	ID         uint `gorm:"primaryKey"`
	Version    string
	Table      string
	ColumnName string
	OldType    string
	NewType    string
	ChangedAt  time.Time `gorm:"autoCreateTime"`
}

func (FieldHistory) TableName() string {
	return "schema_field_history"
}

// EnsureFieldHistoryTable ensures the schema_field_history table exists.
func EnsureFieldHistoryTable(db *gorm.DB) error {
	return db.AutoMigrate(&FieldHistory{})
}

func logFieldAdd(db *gorm.DB, version, table, column, newType string) {
	entry := FieldHistory{Version: version, Table: table, ColumnName: column, NewType: newType}
	_ = db.Create(&entry).Error
}

func logFieldRemove(db *gorm.DB, version, table, column, oldType string) {
	entry := FieldHistory{Version: version, Table: table, ColumnName: column, OldType: oldType}
	_ = db.Create(&entry).Error
}

func logFieldAlter(db *gorm.DB, version, table, column, fromType, toType string) {
	entry := FieldHistory{Version: version, Table: table, ColumnName: column, OldType: fromType, NewType: toType}
	_ = db.Create(&entry).Error
}
