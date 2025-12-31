package models

func Models() []interface{} {
	return []interface{}{
		&User{},
		&Plan{},
		&Tenant{},
		&TenantUser{},
	}
}
