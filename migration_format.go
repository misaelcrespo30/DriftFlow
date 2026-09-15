package driftflow

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	migrationUpMarker   = "-- +migrate Up"
	migrationDownMarker = "-- +migrate Down"
)

// normalizeMigrationBytes canonicalizes migration file bytes before hashing.
// Git autocrlf / editors on Windows often turn LF into CRLF; without this,
// manifest.lock.json hashes break across machines even when SQL is identical.
func normalizeMigrationBytes(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	// Strip UTF-8 BOM if present.
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	b = bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
	return b
}

func migrationFileChecksum(b []byte) string {
	return sha256Hex(normalizeMigrationBytes(b))
}

// checksumMatchesStored accepts the canonical (LF-normalized) hash and the
// legacy raw-byte hash so DBs / manifests created before normalization keep working.
func checksumMatchesStored(stored string, raw []byte) bool {
	if stored == "" {
		return false
	}
	if strings.EqualFold(stored, migrationFileChecksum(raw)) {
		return true
	}
	return strings.EqualFold(stored, sha256Hex(raw))
}

func normalizeMigrationSection(sql string) string {
	return strings.TrimSpace(sql)
}

func formatMigrationFile(upSQL, downSQL string) string {
	up := normalizeMigrationSection(upSQL)
	down := normalizeMigrationSection(downSQL)
	return fmt.Sprintf("%s\n%s\n\n%s\n%s\n", migrationUpMarker, up, migrationDownMarker, down)
}

func splitMigrationSections(contents string) (string, string, error) {
	var (
		upLines   []string
		downLines []string
		seenUp    bool
		seenDown  bool
		state     string
	)

	scanner := bufio.NewScanner(strings.NewReader(contents))
	for scanner.Scan() {
		line := scanner.Text()
		switch strings.TrimSpace(line) {
		case migrationUpMarker:
			if seenUp || seenDown {
				return "", "", fmt.Errorf("unexpected %s marker", migrationUpMarker)
			}
			seenUp = true
			state = "up"
			continue
		case migrationDownMarker:
			if !seenUp || seenDown {
				return "", "", fmt.Errorf("unexpected %s marker", migrationDownMarker)
			}
			seenDown = true
			state = "down"
			continue
		}

		switch state {
		case "up":
			upLines = append(upLines, line)
		case "down":
			downLines = append(downLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if !seenUp || !seenDown {
		return "", "", fmt.Errorf("migration file missing required markers")
	}
	return normalizeMigrationSection(strings.Join(upLines, "\n")),
		normalizeMigrationSection(strings.Join(downLines, "\n")),
		nil
}

func readMigrationSections(path string) (string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return splitMigrationSections(string(normalizeMigrationBytes(b)))
}

func readMigrationFile(path string) (string, string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", err
	}
	// Parse markers from normalized text so CRLF checkouts still split cleanly.
	upSQL, downSQL, err := splitMigrationSections(string(normalizeMigrationBytes(b)))
	if err != nil {
		return "", "", "", err
	}
	return upSQL, downSQL, migrationFileChecksum(b), nil
}

func writeMigrationFile(dir, baseName, upSQL, downSQL string) error {
	path := filepath.Join(dir, baseName+".sql")

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("migration already exists (refusing to overwrite): %s", path)
	}

	content := formatMigrationFile(upSQL, downSQL)
	return os.WriteFile(path, []byte(content), 0o644)
}
