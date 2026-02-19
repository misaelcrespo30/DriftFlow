package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestConfirmDropDatabase(t *testing.T) {
	cmd := &cobra.Command{}
	in := bytes.NewBufferString("tenant_db\n")
	out := &bytes.Buffer{}
	cmd.SetIn(in)
	cmd.SetOut(out)

	ok, err := confirmDropDatabase(cmd, "tenant_db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected confirmation to pass")
	}
}

func TestConfirmDropDatabaseAborted(t *testing.T) {
	cmd := &cobra.Command{}
	in := bytes.NewBufferString("other_db\n")
	out := &bytes.Buffer{}
	cmd.SetIn(in)
	cmd.SetOut(out)

	ok, err := confirmDropDatabase(cmd, "tenant_db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected confirmation to fail")
	}
}

func TestConfirmRestore(t *testing.T) {
	cmd := &cobra.Command{}
	in := bytes.NewBufferString("y\n")
	out := &bytes.Buffer{}
	cmd.SetIn(in)
	cmd.SetOut(out)

	ok, err := confirmRestore(cmd, "tenant_db", "backup.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected restore confirmation to pass")
	}
}

func TestInitDBBackupRequiresOutput(t *testing.T) {
	driver = "postgres"
	dsn = "postgres://user:pass@localhost:5432/app_db?sslmode=disable"

	cmd := newInitDBBackupCommand()
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing --output error")
	}
}
