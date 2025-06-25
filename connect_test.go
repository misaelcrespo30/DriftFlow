package driftflow

import (
	"testing"
)

func TestConnectToDBSQLite(t *testing.T) {
	db, err := ConnectToDB("file::memory:?cache=shared", "sqlite")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if db == nil {
		t.Fatalf("expected db instance")
	}
}

func TestConnectToDBUnsupported(t *testing.T) {
	if _, err := ConnectToDB("", "foo"); err == nil {
		t.Fatalf("expected error for unsupported driver")
	}
}

func TestConnectToDBEnv(t *testing.T) {
	t.Setenv("DSN", "file::memory:?cache=shared")
	t.Setenv("DB_TYPE", "sqlite")
	db, err := ConnectToDB("", "")
	if err != nil {
		t.Fatalf("connect with env: %v", err)
	}
	if db == nil {
		t.Fatalf("expected db instance")
	}
}
