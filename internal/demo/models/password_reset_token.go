package models

import "time"

const TableNamePasswordResetTokens = "password_reset_tokens"

type PasswordResetToken struct {
	ID        string     `gorm:"column:id;size:36;primaryKey" json:"id"`
	UserID    string     `gorm:"column:user_id;size:36;not null;index:ix_prt_user_created,priority:1" json:"user_id"`
	TokenHash string     `gorm:"column:token_hash;size:255;not null;uniqueIndex;index:ix_prt_token_active,priority:1" json:"token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null;index:ix_prt_expires_at" json:"expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at;index:ix_prt_token_active,priority:2" json:"used_at,omitempty"`
	RequestIP string     `gorm:"column:request_ip;size:64;index:ix_prt_request_ip" json:"request_ip"`
	UserAgent string     `gorm:"column:user_agent;size:255" json:"user_agent"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;index:ix_prt_user_created,priority:2" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (*PasswordResetToken) TableName() string {
	return TableNamePasswordResetTokens
}
