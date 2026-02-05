package models

func Models() []interface{} {
	return []interface{}{
		&User{},
		&Plan{},
		&Tenant{},
		&Service{},
		&TenantDomain{},
		&TenantUser{},
		&TenantService{},
		&BackupCode{},
		&PasswordResetToken{},
		&OIDCClient{},
		&OIDCSettings{},
	}
}
