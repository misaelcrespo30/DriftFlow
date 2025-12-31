package models

const TableNameTenantUser = "TenantUsers"

type TenantUser struct {
	ID           string `gorm:"column:id;size:36;primaryKey" json:"id"`
	UserID       string `gorm:"column:user_id;size:36;not null" json:"user_id"`
	TenantID     string `gorm:"column:tenant_id;size:36;not null" json:"tenant_id"`
	Relationship int64  `gorm:"column:relationship;not null" json:"relationship"`

	IsActive       bool `gorm:"column:is_active;not null;default:false" json:"is_active"`
	IsDefault      bool `gorm:"column:is_default;not null;default:false" json:"is_default"`
	OriginatedUser bool `gorm:"column:originated_user;not null;default:false" json:"originated_user"`

	ExternalIdentityID string `gorm:"column:external_identity_id;size:100;not null" json:"external_identity_id"`

	User   *User   `gorm:"foreignKey:UserID;references:UserID" json:"user,omitempty"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantID" json:"tenant,omitempty"`
}

// TableName TenantUser's table name
func (*TenantUser) TableName() string {
	return TableNameTenantUser
}

/*type TenantUser struct {
	ID             uint           `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserId         string         `gorm:"type:uuid;not null;column:user_id" json:"user_id"`
	TenantID       string         `gorm:"type:varchar(128);not null;column:tenant_id" json:"tenant_id" validate:"required"`
	Relationship   int            `gorm:"default:0" json:"relationship"`
	Active         bool           `gorm:"default:false" json:"active"`
	IsDefault      bool           `gorm:"default:false" json:"is_default"`
	OriginatedUser bool           `gorm:"default:false" json:"originated_user"`
	ExternalUserID string         `gorm:"type:uuid;type:varchar(450)" json:"external_user_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relaciones activadas para Preload
	User   *User   `gorm:"foreignKey:UserId;references:UserId" json:"user,omitempty"`
	Tenant *Tenant `gorm:"foreignKey:TenantID;references:TenantId" json:"tenant,omitempty"`
}*/
