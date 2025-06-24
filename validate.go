package schemaflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validate checks migration files for common issues such as duplicated names
// or missing down migration files.
func Validate(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// track migration versions (prefix before first underscore)
	seen := make(map[string]struct{})
	duplicates := []string{}
	missingDown := []string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".up.sql")
		ver := base
		if idx := strings.Index(base, "_"); idx != -1 {
			ver = base[:idx]
		}
		if _, ok := seen[ver]; ok {
			duplicates = append(duplicates, ver)
			continue
		}
		seen[ver] = struct{}{}
		if _, err := os.Stat(filepath.Join(dir, base+".down.sql")); os.IsNotExist(err) {
			missingDown = append(missingDown, base)
		}
	}
	if len(duplicates) > 0 || len(missingDown) > 0 {
		var sb strings.Builder
		if len(duplicates) > 0 {
			sb.WriteString("duplicate migrations: ")
			sb.WriteString(strings.Join(duplicates, ", "))
		}
		if len(missingDown) > 0 {
			if sb.Len() > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString("missing down files: ")
			sb.WriteString(strings.Join(missingDown, ", "))
		}
		return fmt.Errorf(sb.String())
	}
	return nil
}
