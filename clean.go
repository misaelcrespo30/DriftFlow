package driftflow

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"gorm.io/gorm"
)

const (
	migrationsHistoryTable = "migrations_history"
	schemaMigrationsTable  = "schema_migrations"
)

// CleanOptions controls how DriftFlow truncates data without dropping tables.
type CleanOptions struct {
	DSN            string
	Driver         string
	Schema         string
	IncludePattern string
	ExcludePattern string
	KeepMigrations bool
	DryRun         bool
}

// CleanSummary describes the outcome of a clean operation.
type CleanSummary struct {
	Dialect        string
	Database       string
	Schema         string
	TablesAffected int
	Method         string
	DryRun         bool
	Statements     []string
}

// Clean truncates all data in the target database/schema without dropping tables.
func Clean(db *gorm.DB, opts CleanOptions) (CleanSummary, error) {
	dialect := strings.ToLower(db.Dialector.Name())
	if dialect == "" {
		return CleanSummary{}, errors.New("unable to detect database dialect")
	}

	database := ""
	if database == "" {
		var err error
		database, err = databaseFromDSNOrCurrent(db, dialect, opts.DSN)
		if err != nil {
			return CleanSummary{}, err
		}
	}

	schema := strings.TrimSpace(opts.Schema)
	if schema == "" {
		schema = defaultSchemaForDialect(dialect, database)
	}

	summary := CleanSummary{
		Dialect:  dialect,
		Database: database,
		Schema:   schema,
		DryRun:   opts.DryRun,
	}

	var tables []string
	var err error
	switch dialect {
	case "postgres":
		tables, err = listPostgresTables(db, schema)
	case "mysql":
		tables, err = listMySQLTables(db, schema)
	case "sqlserver":
		tables, err = listMSSQLTables(db, schema)
	default:
		return CleanSummary{}, fmt.Errorf("unsupported dialect: %s", dialect)
	}
	if err != nil {
		return CleanSummary{}, err
	}

	tables = filterTables(tables, opts.IncludePattern, opts.ExcludePattern, opts.KeepMigrations)
	summary.TablesAffected = len(tables)
	if len(tables) == 0 {
		summary.Method = "none"
		return summary, nil
	}

	switch dialect {
	case "postgres":
		summary, err = cleanPostgres(db, summary, tables)
	case "mysql":
		summary, err = cleanMySQL(db, summary, tables)
	case "sqlserver":
		summary, err = cleanMSSQL(db, summary, tables)
	}
	if err != nil {
		return CleanSummary{}, err
	}
	return summary, nil
}

func filterTables(tables []string, includePattern, excludePattern string, keepMigrations bool) []string {
	filtered := make([]string, 0, len(tables))
	for _, table := range tables {
		if keepMigrations && isMigrationTable(table) {
			continue
		}
		if includePattern != "" && !matchTablePattern(includePattern, table) {
			continue
		}
		if excludePattern != "" && matchTablePattern(excludePattern, table) {
			continue
		}
		filtered = append(filtered, table)
	}
	return filtered
}

func isMigrationTable(table string) bool {
	return table == migrationsHistoryTable || table == schemaMigrationsTable
}

func matchTablePattern(pattern, table string) bool {
	matched, err := path.Match(pattern, table)
	if err != nil {
		return false
	}
	return matched
}

func cleanPostgres(db *gorm.DB, summary CleanSummary, tables []string) (CleanSummary, error) {
	qualified := make([]string, 0, len(tables))
	for _, table := range tables {
		qualified = append(qualified, quotePostgresIdent(summary.Schema)+"."+quotePostgresIdent(table))
	}
	stmt := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(qualified, ", "))
	summary.Method = "truncate"
	summary.Statements = []string{stmt}
	if summary.DryRun {
		return summary, nil
	}
	if err := db.Exec(stmt).Error; err != nil {
		return summary, err
	}
	return summary, nil
}

func cleanMySQL(db *gorm.DB, summary CleanSummary, tables []string) (CleanSummary, error) {
	method := "truncate"
	statements := []string{"SET FOREIGN_KEY_CHECKS=0"}
	for _, table := range tables {
		qualified := quoteMySQLIdent(summary.Schema) + "." + quoteMySQLIdent(table)
		statements = append(statements, fmt.Sprintf("TRUNCATE TABLE %s", qualified))
	}
	statements = append(statements, "SET FOREIGN_KEY_CHECKS=1")
	if summary.DryRun {
		summary.Method = method
		summary.Statements = statements
		return summary, nil
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS=0").Error; err != nil {
		return summary, err
	}
	for _, table := range tables {
		qualified := quoteMySQLIdent(summary.Schema) + "." + quoteMySQLIdent(table)
		truncateStmt := fmt.Sprintf("TRUNCATE TABLE %s", qualified)
		if err := db.Exec(truncateStmt).Error; err != nil {
			deleteStmt := fmt.Sprintf("DELETE FROM %s", qualified)
			if err := db.Exec(deleteStmt).Error; err != nil {
				return summary, err
			}
			method = "truncate/delete"
		}
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS=1").Error; err != nil {
		return summary, err
	}

	summary.Method = method
	return summary, nil
}

func cleanMSSQL(db *gorm.DB, summary CleanSummary, tables []string) (CleanSummary, error) {
	statements := make([]string, 0, len(tables))
	for _, table := range tables {
		statements = append(statements, fmt.Sprintf("TRUNCATE TABLE %s", quoteMSSQLIdentWithSchema(summary.Schema, table)))
	}
	if summary.DryRun {
		summary.Method = "truncate"
		summary.Statements = statements
		return summary, nil
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return cleanMSSQLDelete(db, summary, tables)
		}
	}
	summary.Method = "truncate"
	return summary, nil
}

func cleanMSSQLDelete(db *gorm.DB, summary CleanSummary, tables []string) (CleanSummary, error) {
	if err := db.Exec(`EXEC sp_msforeachtable "ALTER TABLE ? NOCHECK CONSTRAINT all"`).Error; err != nil {
		return summary, err
	}
	for _, table := range tables {
		qualified := quoteMSSQLIdentWithSchema(summary.Schema, table)
		deleteStmt := fmt.Sprintf("DELETE FROM %s", qualified)
		if err := db.Exec(deleteStmt).Error; err != nil {
			return summary, err
		}
		reseedStmt := fmt.Sprintf("DBCC CHECKIDENT ('%s.%s', RESEED, 0)", summary.Schema, table)
		if err := db.Exec(reseedStmt).Error; err != nil {
			return summary, err
		}
	}
	if err := db.Exec(`EXEC sp_msforeachtable "ALTER TABLE ? WITH CHECK CHECK CONSTRAINT all"`).Error; err != nil {
		return summary, err
	}
	summary.Method = "delete"
	return summary, nil
}
