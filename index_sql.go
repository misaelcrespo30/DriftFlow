package driftflow

import (
	"fmt"
	"sort"
	"strings"
)

func appendIndexSQL(baseSQL string, table string, indexes []IndexDefinition, engine string) string {
	if len(indexes) == 0 {
		return baseSQL
	}
	var parts []string
	for _, idx := range indexes {
		parts = append(parts, createIndexSQL(table, idx, engine))
	}
	return strings.TrimSpace(baseSQL) + "\n" + strings.Join(parts, "\n")
}

func appendIndexChanges(upSQL, downSQL, table string, added, removed []IndexDefinition, engine string) (string, string) {
	if len(added) == 0 && len(removed) == 0 {
		return upSQL, downSQL
	}
	added = cloneIndexes(added)
	removed = cloneIndexes(removed)
	sort.Slice(added, func(i, j int) bool { return added[i].Name < added[j].Name })
	sort.Slice(removed, func(i, j int) bool { return removed[i].Name < removed[j].Name })

	var upParts []string
	var downParts []string
	if upSQL != "" {
		upParts = append(upParts, upSQL)
	}
	if downSQL != "" {
		downParts = append(downParts, downSQL)
	}

	for _, idx := range removed {
		upParts = append(upParts, dropIndexSQL(table, idx.Name, engine))
		downParts = append([]string{createIndexSQL(table, idx, engine)}, downParts...)
	}
	for _, idx := range added {
		upParts = append(upParts, createIndexSQL(table, idx, engine))
		downParts = append([]string{dropIndexSQL(table, idx.Name, engine)}, downParts...)
	}

	return strings.Join(upParts, "\n"), strings.Join(downParts, "\n")
}

func diffIndexes(prev, next []IndexDefinition) (added []IndexDefinition, removed []IndexDefinition) {
	prevMap := make(map[string]IndexDefinition, len(prev))
	for _, idx := range prev {
		prevMap[idx.Name] = normalizeIndex(idx)
	}
	nextMap := make(map[string]IndexDefinition, len(next))
	for _, idx := range next {
		nextMap[idx.Name] = normalizeIndex(idx)
	}

	for name, prevIdx := range prevMap {
		nextIdx, ok := nextMap[name]
		if !ok || !indexesEqual(prevIdx, nextIdx) {
			removed = append(removed, prevIdx)
		}
	}
	for name, nextIdx := range nextMap {
		prevIdx, ok := prevMap[name]
		if !ok || !indexesEqual(prevIdx, nextIdx) {
			added = append(added, nextIdx)
		}
	}

	return added, removed
}

func createIndexSQL(table string, idx IndexDefinition, engine string) string {
	cols := make([]string, len(idx.Columns))
	for i, col := range idx.Columns {
		cols[i] = quoteIdent(engine, col)
	}
	unique := ""
	if idx.Unique {
		unique = "UNIQUE "
	}
	if normalizeEngine(engine) == "postgres" {
		stmt := fmt.Sprintf("CREATE %sINDEX IF NOT EXISTS %s ON %s (%s)", unique, quoteIdent(engine, idx.Name), quoteIdent(engine, table), strings.Join(cols, ", "))
		if strings.TrimSpace(idx.Where) != "" {
			stmt += " WHERE " + idx.Where
		}
		return stmt + ";"
	}

	stmt := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)", unique, quoteIdent(engine, idx.Name), quoteIdent(engine, table), strings.Join(cols, ", "))
	if strings.TrimSpace(idx.Where) != "" {
		stmt += " WHERE " + idx.Where
	}
	return stmt + ";"
}

func dropIndexSQL(table string, name string, engine string) string {
	switch normalizeEngine(engine) {
	case "mysql", "sqlserver", "mssql":
		return fmt.Sprintf("DROP INDEX %s ON %s;", quoteIdent(engine, name), quoteIdent(engine, table))
	default:
		return fmt.Sprintf("DROP INDEX %s;", quoteIdent(engine, name))
	}
}

func normalizeIndex(idx IndexDefinition) IndexDefinition {
	cols := append([]string{}, idx.Columns...)
	return IndexDefinition{
		Name:    idx.Name,
		Columns: cols,
		Unique:  idx.Unique,
		Where:   normalizeWhere(idx.Where),
	}
}

func normalizeWhere(where string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(where)), " "))
}

func indexesEqual(a, b IndexDefinition) bool {
	if a.Name != b.Name || a.Unique != b.Unique || normalizeWhere(a.Where) != normalizeWhere(b.Where) {
		return false
	}
	if len(a.Columns) != len(b.Columns) {
		return false
	}
	for i := range a.Columns {
		if a.Columns[i] != b.Columns[i] {
			return false
		}
	}
	return true
}
