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

func TestConfirmRebootstrapRequiresDESTROY(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("yes\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := confirmRebootstrap(cmd, false, false, ""); err == nil {
		t.Fatal("expected abort when input is not DESTROY")
	}

	cmd = &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("DESTROY\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := confirmRebootstrap(cmd, false, false, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfirmRebootstrapCIFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	if err := confirmRebootstrap(cmd, true, false, ""); err == nil {
		t.Fatal("expected --i-know-what-im-doing to require backup-output or skip-backup")
	}
	if err := confirmRebootstrap(cmd, true, true, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPromptBackupBeforeReset(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("\n"))
	cmd.SetOut(&bytes.Buffer{})
	ok, err := promptBackupBeforeReset(cmd)
	if err != nil || !ok {
		t.Fatalf("empty answer should default to yes, got ok=%v err=%v", ok, err)
	}

	cmd = &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("n\n"))
	cmd.SetOut(&bytes.Buffer{})
	ok, err = promptBackupBeforeReset(cmd)
	if err != nil || ok {
		t.Fatalf("n should decline backup, got ok=%v err=%v", ok, err)
	}
}

func TestConfirmDataLoss(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("no\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := confirmDataLoss(cmd); err == nil {
		t.Fatal("expected abort")
	}
	cmd = &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("I ACCEPT DATA LOSS\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := confirmDataLoss(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsProductionEnv(t *testing.T) {
	t.Setenv("ENV", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("ENVIRONMENT", "")
	if isProductionEnv() {
		t.Fatal("expected non-production with empty env")
	}
	t.Setenv("APP_ENV", "production")
	if !isProductionEnv() {
		t.Fatal("expected APP_ENV=production to be production")
	}
	t.Setenv("APP_ENV", "")
	t.Setenv("ENVIRONMENT", "prod")
	if !isProductionEnv() {
		t.Fatal("expected ENVIRONMENT=prod to be production")
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
