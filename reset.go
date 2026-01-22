package driftflow

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"gorm.io/gorm"
)

// ResetOptions controls how DriftFlow clears a database in development.
type ResetOptions struct {
	DSN      string
	Driver   string
	Schema   string
	Database string
}

// ResetSummary describes the outcome of a reset operation.
type ResetSummary struct {
	Dialect           string
	Database          string
	Schema            string
	TablesDropped     int
	RecreatedSchema   bool
	RecreatedDatabase bool
}

// Reset removes all tables in the target database/schema based on the detected dialect.
func Reset(db *gorm.DB, opts ResetOptions) (ResetSummary, error) {
	dialect := strings.ToLower(db.Dialector.Name())
	if dialect == "" {
		return ResetSummary{}, errors.New("unable to detect database dialect")
	}

	database := strings.TrimSpace(opts.Database)
	if database == "" {
		var err error
		database, err = databaseFromDSNOrCurrent(db, dialect, opts.DSN)
		if err != nil {
			return ResetSummary{}, err
		}
	}

	schema := strings.TrimSpace(opts.Schema)
	if schema == "" {
		schema = defaultSchemaForDialect(dialect, database)
	}

	summary := ResetSummary{
		Dialect:  dialect,
		Database: database,
		Schema:   schema,
	}

	var err error
	switch dialect {
	case "postgres":
		summary, err = resetPostgres(db, opts, summary)
	case "mysql":
		summary, err = resetMySQL(db, opts, summary)
	case "sqlserver":
		summary, err = resetMSSQL(db, opts, summary)
	default:
		return ResetSummary{}, fmt.Errorf("unsupported dialect: %s", dialect)
	}
	if err != nil {
		return ResetSummary{}, err
	}
	return summary, nil
}

func defaultSchemaForDialect(dialect, database string) string {
	switch dialect {
	case "postgres":
		return "public"
	case "sqlserver":
		return "dbo"
	case "mysql":
		return database
	default:
		return ""
	}
}

func databaseFromDSNOrCurrent(db *gorm.DB, dialect, dsn string) (string, error) {
	if dsn != "" {
		if name := databaseFromDSN(dialect, dsn); name != "" {
			return name, nil
		}
	}

	switch dialect {
	case "postgres":
		var name string
		if err := db.Raw("SELECT current_database()").Scan(&name).Error; err != nil {
			return "", err
		}
		return name, nil
	case "mysql":
		var name string
		if err := db.Raw("SELECT DATABASE()").Scan(&name).Error; err != nil {
			return "", err
		}
		return name, nil
	case "sqlserver":
		var name string
		if err := db.Raw("SELECT DB_NAME()").Scan(&name).Error; err != nil {
			return "", err
		}
		return name, nil
	default:
		return "", fmt.Errorf("unsupported dialect: %s", dialect)
	}
}

func databaseFromDSN(dialect, dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	path := strings.TrimPrefix(parsed.Path, "/")
	switch dialect {
	case "postgres", "mysql":
		return path
	case "sqlserver":
		return parsed.Query().Get("database")
	default:
		return ""
	}
}

func resetPostgres(db *gorm.DB, opts ResetOptions, summary ResetSummary) (ResetSummary, error) {
	tables, err := listPostgresTables(db, summary.Schema)
	if err != nil {
		return summary, err
	}
	summary.TablesDropped = len(tables)

	schemaIdent := quotePostgresIdent(summary.Schema)
	if err := db.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaIdent)).Error; err != nil {
		return summary, err
	}
	if err := db.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaIdent)).Error; err != nil {
		return summary, err
	}
	if err := db.Exec(fmt.Sprintf("GRANT USAGE, CREATE ON SCHEMA %s TO PUBLIC", schemaIdent)).Error; err != nil {
		return summary, err
	}
	summary.RecreatedSchema = true
	return summary, nil
}

func resetMySQL(db *gorm.DB, opts ResetOptions, summary ResetSummary) (ResetSummary, error) {
	database := summary.Database
	if database == "" {
		return summary, errors.New("database name is required for MySQL reset")
	}
	tables, err := listMySQLTables(db, database)
	if err != nil {
		return summary, err
	}
	summary.TablesDropped = len(tables)

	if err := dropAndCreateMySQLDatabase(db, opts, database); err == nil {
		summary.RecreatedDatabase = true
		return summary, nil
	}

	if len(tables) == 0 {
		return summary, nil
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS=0").Error; err != nil {
		return summary, err
	}

	dropStmt := fmt.Sprintf("DROP TABLE %s", strings.Join(quoteMySQLIdents(tables), ", "))
	if err := db.Exec(dropStmt).Error; err != nil {
		return summary, err
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS=1").Error; err != nil {
		return summary, err
	}

	return summary, nil
}

func dropAndCreateMySQLDatabase(db *gorm.DB, opts ResetOptions, database string) error {
	dbIdent := quoteMySQLIdent(database)
	if err := db.Exec(fmt.Sprintf("DROP DATABASE %s", dbIdent)).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbIdent)).Error; err != nil {
		return err
	}
	if opts.DSN != "" && opts.Driver != "" {
		if _, err := ConnectToDB(opts.DSN, opts.Driver); err != nil {
			return err
		}
	}
	return nil
}

func resetMSSQL(db *gorm.DB, opts ResetOptions, summary ResetSummary) (ResetSummary, error) {
	tables, err := listMSSQLTables(db, summary.Schema)
	if err != nil {
		return summary, err
	}
	summary.TablesDropped = len(tables)

	constraints, err := listMSSQLForeignKeys(db, summary.Schema)
	if err != nil {
		return summary, err
	}

	for _, fk := range constraints {
		stmt := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", quoteMSSQLIdentWithSchema(fk.SchemaName, fk.TableName), quoteMSSQLIdent(fk.ConstraintName))
		if err := db.Exec(stmt).Error; err != nil {
			return summary, err
		}
	}

	for _, table := range tables {
		stmt := fmt.Sprintf("DROP TABLE %s", quoteMSSQLIdentWithSchema(summary.Schema, table))
		if err := db.Exec(stmt).Error; err != nil {
			return summary, err
		}
	}

	return summary, nil
}

type foreignKeyRow struct {
	SchemaName     string `gorm:"column:schema_name"`
	TableName      string `gorm:"column:table_name"`
	ConstraintName string `gorm:"column:constraint_name"`
}

func listPostgresTables(db *gorm.DB, schema string) ([]string, error) {
	rows := []struct {
		Name string `gorm:"column:table_name"`
	}{}
	err := db.Raw(`
SELECT table_name
FROM information_schema.tables
WHERE table_schema = ?
  AND table_type = 'BASE TABLE'
`, schema).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return extractTableNames(rows), nil
}

func listMySQLTables(db *gorm.DB, database string) ([]string, error) {
	rows := []struct {
		Name string `gorm:"column:table_name"`
	}{}
	err := db.Raw(`
SELECT table_name
FROM information_schema.tables
WHERE table_schema = ?
  AND table_type = 'BASE TABLE'
`, database).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return extractTableNames(rows), nil
}

func listMSSQLTables(db *gorm.DB, schema string) ([]string, error) {
	rows := []struct {
		Name string `gorm:"column:table_name"`
	}{}
	err := db.Raw(`
SELECT TABLE_NAME AS table_name
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = ?
  AND TABLE_TYPE = 'BASE TABLE'
`, schema).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return extractTableNames(rows), nil
}

func listMSSQLForeignKeys(db *gorm.DB, schema string) ([]foreignKeyRow, error) {
	rows := []foreignKeyRow{}
	err := db.Raw(`
SELECT s.name AS schema_name,
       t.name AS table_name,
       fk.name AS constraint_name
FROM sys.foreign_keys fk
JOIN sys.tables t ON fk.parent_object_id = t.object_id
JOIN sys.schemas s ON t.schema_id = s.schema_id
WHERE s.name = ?
`, schema).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func extractTableNames(rows []struct{ Name string }) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Name != "" {
			names = append(names, row.Name)
		}
	}
	return names
}

func quotePostgresIdent(name string) string {
	return "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\""
}

func quoteMySQLIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func quoteMySQLIdents(names []string) []string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, quoteMySQLIdent(name))
	}
	return quoted
}

func quoteMSSQLIdent(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}

func quoteMSSQLIdentWithSchema(schema, name string) string {
	return quoteMSSQLIdent(schema) + "." + quoteMSSQLIdent(name)
}
