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
	return buildAlterSQLWithEngine(table, "", "postgres", prevCols, nextCols, order, added, removed, altered)
}

// buildAlterSQLWithEngine emits ADD/DROP/ALTER and UNIQUE-constraint repairs.
// tableQuoted is the dialect-quoted table ident; tableRaw is the unquoted name
// used for Postgres default UNIQUE constraint naming ({table}_{column}_key).
func buildAlterSQLWithEngine(
	tableQuoted string,
	tableRaw string,
	engine string,
	prevCols map[string]string,
	nextCols map[string]string,
	order []string,
	added map[string]string,
	removed map[string]string,
	altered map[string]ColAlter,
) (up string, down string) {

	var upParts []string
	var downParts []string
	engine = normalizeEngine(engine)
	if tableRaw == "" {
		tableRaw = strings.Trim(tableQuoted, `"'[]`)
	}

	// ADD (orden estable)
	addKeys := make([]string, 0, len(added))
	for k := range added {
		addKeys = append(addKeys, k)
	}
	sort.Strings(addKeys)
	for _, col := range addKeys {
		upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tableQuoted, quoteIdent(engine, col), nextCols[col]))
		downParts = append([]string{fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableQuoted, quoteIdent(engine, col))}, downParts...)
	}

	// DROP
	remKeys := make([]string, 0, len(removed))
	for k := range removed {
		remKeys = append(remKeys, k)
	}
	sort.Strings(remKeys)
	for _, col := range remKeys {
		upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableQuoted, quoteIdent(engine, col)))
		downParts = append([]string{fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tableQuoted, quoteIdent(engine, col), prevCols[col])}, downParts...)
	}

	// ALTER (type / nullability / unique constraint)
	altKeys := make([]string, 0, len(altered))
	for k := range altered {
		altKeys = append(altKeys, k)
	}
	sort.Strings(altKeys)
	for _, col := range altKeys {
		a := altered[col]
		prevUnique := defHasUniqueToken(a.From)
		nextUnique := defHasUniqueToken(a.To)
		baseFrom := stripUniqueToken(normalizeDef(a.From))
		baseTo := stripUniqueToken(normalizeDef(a.To))

		if prevUnique && !nextUnique {
			cname := defaultUniqueConstraintName(tableRaw, col)
			upParts = append(upParts, fmt.Sprintf(
				"ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;",
				tableQuoted, quoteIdent(engine, cname),
			))
			downParts = append([]string{fmt.Sprintf(
				"ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);",
				tableQuoted, quoteIdent(engine, cname), quoteIdent(engine, col),
			)}, downParts...)
		}
		if !prevUnique && nextUnique {
			cname := defaultUniqueConstraintName(tableRaw, col)
			upParts = append(upParts, fmt.Sprintf(
				"ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (%s);",
				tableQuoted, quoteIdent(engine, cname), quoteIdent(engine, col),
			))
			downParts = append([]string{fmt.Sprintf(
				"ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;",
				tableQuoted, quoteIdent(engine, cname),
			)}, downParts...)
		}

		if baseFrom != baseTo {
			// Preserve legacy ALTER TYPE emission for non-uniqueness changes.
			upParts = append(upParts, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", tableQuoted, quoteIdent(engine, col), a.To))
			downParts = append([]string{fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;", tableQuoted, quoteIdent(engine, col), a.From)}, downParts...)
		}
	}

	return strings.Join(upParts, "\n"), strings.Join(downParts, "\n")
}

func defaultUniqueConstraintName(table, column string) string {
	return table + "_" + column + "_key"
}

func defHasUniqueToken(def string) bool {
	for _, tok := range strings.Fields(strings.ToLower(def)) {
		if tok == "unique" {
			return true
		}
	}
	return false
}

func stripUniqueToken(def string) string {
	parts := strings.Fields(def)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.EqualFold(p, "unique") {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, " ")
}
