package driftflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeMigrationBytesStripsCRLFAndBOM(t *testing.T) {
	raw := []byte("\xEF\xBB\xBF-- +migrate Up\r\nSELECT 1;\r\n\r\n-- +migrate Down\r\nSELECT 0;\r\n")
	got := normalizeMigrationBytes(raw)
	want := []byte("-- +migrate Up\nSELECT 1;\n\n-- +migrate Down\nSELECT 0;\n")
	if string(got) != string(want) {
		t.Fatalf("normalize mismatch:\n got %q\nwant %q", got, want)
	}
}

func TestMigrationChecksumIgnoresLineEndings(t *testing.T) {
	lf := []byte("-- +migrate Up\nSELECT 1;\n\n-- +migrate Down\nSELECT 0;\n")
	crlf := []byte("-- +migrate Up\r\nSELECT 1;\r\n\r\n-- +migrate Down\r\nSELECT 0;\r\n")

	lfHash := migrationFileChecksum(lf)
	crlfHash := migrationFileChecksum(crlf)
	if lfHash != crlfHash {
		t.Fatalf("expected identical checksums for LF and CRLF, got %s vs %s", lfHash, crlfHash)
	}
	if lfHash == sha256Hex(crlf) {
		t.Fatalf("canonical checksum should differ from raw CRLF hash")
	}
	if !checksumMatchesStored(lfHash, crlf) {
		t.Fatalf("checksumMatchesStored should accept LF hash against CRLF file bytes")
	}
	if !checksumMatchesStored(sha256Hex(crlf), crlf) {
		t.Fatalf("checksumMatchesStored should accept legacy raw CRLF hash")
	}
}

func TestManifestAcceptsCRLFCheckout(t *testing.T) {
	dir := t.TempDir()
	name := "2026_01_01_000001_create_demo_table.sql"
	lf := []byte("-- +migrate Up\nCREATE TABLE demo (id int);\n\n-- +migrate Down\nDROP TABLE demo;\n")
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, lf, 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashMigrationFile(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest := &ManifestLock{
		Version: 1,
		Migrations: []ManifestEntry{{
			Name:       name,
			SQLSHA256:  hash,
			CreatedUTC: "2026-01-01T00:00:00Z",
		}},
	}

	// Simulate git autocrlf checkout on another machine.
	crlf := []byte("-- +migrate Up\r\nCREATE TABLE demo (id int);\r\n\r\n-- +migrate Down\r\nDROP TABLE demo;\r\n")
	if err := os.WriteFile(path, crlf, 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := validateManifest(dir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no manifest issues after CRLF checkout, got %#v", issues)
	}
}
