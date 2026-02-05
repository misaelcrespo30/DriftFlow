package models

import "time"

const TableNameBackupCodes = "user_backup_codes"

type BackupCode struct {
	ID        string     `gorm:"column:id;size:36;primaryKey" json:"id"`
	UserID    string     `gorm:"column:user_id;size:36;not null;index:ix_backup_codes_user_created,priority:1;index:ix_backup_codes_user_code_active,priority:1" json:"user_id"`
	CodeHash  string     `gorm:"column:code_hash;size:200;not null;index:ix_backup_codes_user_code_active,priority:2" json:"-"`
	UsedAt    *time.Time `gorm:"column:used_at;index:ix_backup_codes_user_code_active,priority:3" json:"used_at,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;index:ix_backup_codes_user_created,priority:2" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null" json:"updated_at"`
	User      *User      `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
}

func (*BackupCode) TableName() string {
	return TableNameBackupCodes
}
