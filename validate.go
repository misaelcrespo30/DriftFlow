package driftflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/misaelcrespo30/DriftFlow/config"
)

// Validate checks migration files for common issues such as duplicated names
// or missing down migration files.
func Validate(dir string) error {
	if err := config.ValidateDir(dir); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// track migration versions (prefix before first underscore)
	seen := make(map[string]struct{})
	duplicates := []string{}
	missingDown := []string{}
	namingIssues := []string{}

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

		sqlBytes, err := os.ReadFile(filepath.Join(dir, base+".up.sql"))
		if err != nil {
			return err
		}
		namingIssues = append(namingIssues, checkNamingConventions(string(sqlBytes))...)
	}
	if len(duplicates) > 0 || len(missingDown) > 0 || len(namingIssues) > 0 {
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
		if len(namingIssues) > 0 {
			if sb.Len() > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString("naming issues: ")
			sb.WriteString(strings.Join(namingIssues, ", "))
		}
		return fmt.Errorf("%s", sb.String())
	}
	return nil
}

func isSnakeCase(name string) bool {
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9_]*$`, name)
	return matched
}

func checkNamingConventions(sql string) []string {
	issues := []string{}
	reTable := regexp.MustCompile("(?i)create\\s+table\\s+([\\w\"`]+)\\s*\\(([^;]+)\\)")
	tbls := reTable.FindAllStringSubmatch(sql, -1)
	for _, m := range tbls {
		tbl := strings.Trim(m[1], "`\"")
		if !isSnakeCase(tbl) {
			issues = append(issues, fmt.Sprintf("table %s not snake_case", tbl))
		}
		cols := strings.Split(m[2], ",")
		for _, colLine := range cols {
			fields := strings.Fields(strings.TrimSpace(colLine))
			if len(fields) == 0 {
				continue
			}
			col := strings.Trim(fields[0], "`\"")
			if strings.EqualFold(col, "constraint") || strings.EqualFold(col, "primary") || strings.EqualFold(col, "foreign") {
				continue
			}
			if !isSnakeCase(col) {
				issues = append(issues, fmt.Sprintf("column %s not snake_case", col))
			}
		}
	}
	reAlter := regexp.MustCompile("(?i)alter\\s+table\\s+([\\w\"`]+)\\s+add\\s+column\\s+([\\w\"`]+)")
	alts := reAlter.FindAllStringSubmatch(sql, -1)
	for _, m := range alts {
		tbl := strings.Trim(m[1], "`\"")
		col := strings.Trim(m[2], "`\"")
		if !isSnakeCase(tbl) {
			issues = append(issues, fmt.Sprintf("table %s not snake_case", tbl))
		}
		if !isSnakeCase(col) {
			issues = append(issues, fmt.Sprintf("column %s not snake_case", col))
		}
	}
	return issues
}
