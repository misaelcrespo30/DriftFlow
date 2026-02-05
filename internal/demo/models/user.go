package models

import (
	"gorm.io/gorm"
	"time"
)

const TableNameUser = "users"

type User struct {
	UserID                string         `gorm:"column:user_id;size:36;primaryKey" json:"user_id"`
	Email                 string         `gorm:"column:email;size:100;not null;uniqueIndex;index:ix_users_email_deleted,priority:1" json:"email"`
	UserName              string         `gorm:"column:username;size:100;uniqueIndex;index:ix_users_username_deleted,priority:1" json:"username"`
	FirstName             *string        `gorm:"column:first_name;size:100" json:"first_name,omitempty"`
	LastName              *string        `gorm:"column:last_name;size:100" json:"last_name,omitempty"`
	PasswordHash          string         `gorm:"column:password_hash;size:250;not null" json:"-"`
	TwoFactorEnabled      bool           `gorm:"column:two_factor_enabled;not null;default:false" json:"two_factor_enabled"`
	TwoFactorSetupPending bool           `gorm:"column:two_factor_setup_pending;not null;default:false" json:"two_factor_setup_pending"`
	TwoFactorSecretEnc    *string        `gorm:"column:two_factor_secret_enc" json:"-"`
	TwoFactorLastUsedAt   *time.Time     `gorm:"column:two_factor_last_used_at" json:"two_factor_last_used_at,omitempty"`
	TwoFactorLastUsedStep *int64         `gorm:"column:two_factor_last_used_step" json:"-"`
	AccessFailedCount     int64          `gorm:"column:access_failed_count;not null;default:0" json:"access_failed_count"`
	IsEmailConfirmed      bool           `gorm:"column:is_email_confirmed;not null;default:false;index:ix_users_is_email_confirmed" json:"is_email_confirmed"`
	IsLockoutEnabled      bool           `gorm:"column:is_lockout_enabled;not null;default:false;index:ix_users_is_lockout_enabled" json:"is_lockout_enabled"`
	LockoutEnd            *time.Time     `gorm:"column:lockout_end;index:ix_users_lockout_end" json:"lockout_end,omitempty"`
	Phone                 *string        `gorm:"column:phone;size:20" json:"phone,omitempty"`
	IsPhoneConfirmed      bool           `gorm:"column:is_phone_confirmed;not null;default:false" json:"is_phone_confirmed"`
	SecurityStamp         *string        `gorm:"column:security_stamp;size:100" json:"security_stamp,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index;index:ix_users_email_deleted,priority:2;index:ix_users_username_deleted,priority:2" json:"-"`
	TenantUsers           []TenantUser   `gorm:"foreignKey:UserID;references:UserID" json:"tenant_users,omitempty"`
}

func (*User) TableName() string {
	return TableNameUser
}
