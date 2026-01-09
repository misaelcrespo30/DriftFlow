package driftflow

import "strings"

func normalizeEngine(engine string) string {
	return strings.ToLower(strings.TrimSpace(engine))
}

func quoteIdent(engine, ident string) string {
	switch normalizeEngine(engine) {
	case "sqlserver", "mssql":
		return "[" + ident + "]"
	case "mysql":
		return "`" + ident + "`"
	default:
		return `"` + ident + `"`
	}
}
