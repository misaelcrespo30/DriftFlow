package driftflow

import "testing"

func TestValidateDBName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "tenant_db_01"},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid chars", input: "tenant-db", wantErr: true},
		{name: "reserved", input: "postgres", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDBName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateDBName(%q) error=%v wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestResolveDBName(t *testing.T) {
	svc := NewDbAdminService("postgres://user:pass@localhost:5432/default_db?sslmode=disable", "postgres")

	name, err := svc.ResolveDBName("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "default_db" {
		t.Fatalf("expected default_db, got %s", name)
	}

	name, err = svc.ResolveDBName("tenant_01")
	if err != nil {
		t.Fatalf("unexpected error with override: %v", err)
	}
	if name != "tenant_01" {
		t.Fatalf("expected tenant_01, got %s", name)
	}

	if _, err = svc.ResolveDBName("template1"); err == nil {
		t.Fatal("expected error for reserved override db")
	}
}

func TestResolveDBNameByDriver(t *testing.T) {
	tests := []struct {
		name   string
		dsn    string
		driver string
		want   string
	}{
		{name: "mysql", dsn: "mysql://user:pass@localhost:3306/mysql_db", driver: "mysql", want: "mysql_db"},
		{name: "sqlserver", dsn: "sqlserver://user:pass@localhost:1433?database=sqlsrv_db", driver: "sqlserver", want: "sqlsrv_db"},
		{name: "mongodb", dsn: "mongodb://user:pass@localhost:27017/mongo_db", driver: "mongodb", want: "mongo_db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDbAdminService(tt.dsn, tt.driver)
			got, err := svc.ResolveDBName("")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestResolvePostgresConnectionString(t *testing.T) {
	svc := NewDbAdminService("postgres://postgres:postgres@postgres:5432/apexbuildr_auth_service?sslmode=disable", "postgres")
	name, err := svc.ResolveDBName("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "apexbuildr_auth_service" {
		t.Fatalf("expected apexbuildr_auth_service, got %s", name)
	}
}
