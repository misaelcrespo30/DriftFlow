package driftflow

import (
	"fmt"
	"sort"
	"strings"
)

func buildAlterSQL(
	table string,
	prevCols map[string]string,
	nextCols map[string]string,
	order []string,
	added map[string]string,
	removed map[string]string,
	altered map[string]ColAlter,
) (up string, down string) {

	var upParts []string
	var downParts []string

	// ADD (orden estable)
	addKeys := make([]string, 0, len(added))
	for k := range added {
		addKeys = append(addKeys, k)
	}
	sort.Strings(addKeys)
	for _, col := range addKeys {
		upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, col, nextCols[col]))
		downParts = append([]string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", table, col)}, downParts...)
	}

	// DROP
	remKeys := make([]string, 0, len(removed))
	for k := range removed {
		remKeys = append(remKeys, k)
	}
	sort.Strings(remKeys)
	for _, col := range remKeys {
		upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", table, col))
		downParts = append([]string{fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, col, prevCols[col])}, downParts...)
	}

	// ALTER
	altKeys := make([]string, 0, len(altered))
	for k := range altered {
		altKeys = append(altKeys, k)
	}
	sort.Strings(altKeys)
	for _, col := range altKeys {
		a := altered[col]
		// Nota: ALTER TYPE es Postgres; si quieres SQL Server también, se hace “por driver” (te lo dejo abajo).
		upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", table, col, a.To))
		downParts = append([]string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", table, col, a.From)}, downParts...)
	}

	return strings.Join(upParts, "\n"), strings.Join(downParts, "\n")
}
