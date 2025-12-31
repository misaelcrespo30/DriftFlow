package models

import (
	"time"
)

const TableNameUser = "Users"

type User struct {
	UserID            string     `gorm:"column:user_id;size:36;primaryKey" json:"user_id"`
	Email             string     `gorm:"column:email;size:100;not null;uniqueIndex" json:"email"`
	UserName          string     `gorm:"column:username;size:100;uniqueIndex" json:"username"`
	PasswordHash      string     `gorm:"column:password_hash;size:250;not null" json:"-"`
	AccessFailedCount int64      `gorm:"column:access_failed_count;not null;default:0" json:"access_failed_count"`
	IsEmailConfirmed  bool       `gorm:"column:is_email_confirmed;not null;default:false" json:"is_email_confirmed"`
	IsLockoutEnabled  bool       `gorm:"column:is_lockout_enabled;not null;default:false" json:"is_lockout_enabled"`
	LockoutEnd        *time.Time `gorm:"column:lockout_end" json:"lockout_end,omitempty"`
	Phone             *string    `gorm:"column:phone;size:20" json:"phone,omitempty"`
	IsPhoneConfirmed  bool       `gorm:"column:is_phone_confirmed;not null;default:false" json:"is_phone_confirmed"`
	SecurityStamp     *string    `gorm:"column:security_stamp;size:100" json:"security_stamp,omitempty"`
	// Relaciones
	TenantUsers []TenantUser `gorm:"foreignKey:UserID;references:UserID" json:"tenant_users,omitempty"`

	Apellido string `gorm:"column:apellido;size:100;uniqueIndex" json:"apellido"`
	Misael   string `gorm:"column:misael;size:100;uniqueIndex" json:"misael"`
}

// TableName User's table name
func (*User) TableName() string {
	return TableNameUser
}

/*type User struct {
	ID                uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId            string         `gorm:"type:string;default:uuid_generate_v4();uniqueIndex" json:"user_id"`
	Email             string         `gorm:"unique;not null;type:varchar(100)" json:"email" validate:"required,email"`
	UserName          string         `gorm:"unique;not null;type:varchar(100)" json:"username" validate:"required"`
	PasswordHash      string         `gorm:"not null;type:varchar(250)" json:"password_hash" validate:"required"`
	Role              string         `gorm:"type:varchar(20)" json:"role"`
	Phone             string         `gorm:"type:varchar(20)" json:"phone,omitempty"`
	PhoneConfirmed    bool           `gorm:"default:false" json:"phone_confirmed"`
	EmailConfirmed    bool           `gorm:"default:false" json:"email_confirmed"`
	AccessFailedCount int            `gorm:"default:0" json:"access_failed_count"`
	LockoutEnabled    bool           `gorm:"default:false" json:"lockout_enabled"`
	LockoutEnd        *time.Time     `gorm:"type:timestamp" json:"lockout_end,omitempty"`
	IsOrganization    bool           `gorm:"default:false" json:"is_organization"`
	SecurityStamp     string         `gorm:"type:varchar(100)" json:"-"`
	TenantUsers       []TenantUser   `gorm:"foreignKey:UserId;references:UserId" json:"tenant_users,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}*/
