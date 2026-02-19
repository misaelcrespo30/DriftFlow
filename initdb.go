package driftflow

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
)

var dbNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

var reservedDBNames = map[string]struct{}{
	"postgres":  {},
	"template0": {},
	"template1": {},
}

// DbAdminService offers safe database administration operations.
type DbAdminService struct {
	dsn    string
	driver string
}

// NewDbAdminService creates a new service using configured DSN and driver.
func NewDbAdminService(dsn, driver string) *DbAdminService {
	return &DbAdminService{
		dsn:    strings.TrimSpace(dsn),
		driver: strings.ToLower(strings.TrimSpace(driver)),
	}
}

// ValidateDBName validates tenant/db names accepted by initdb flags.
func ValidateDBName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("database name is required")
	}
	if !dbNamePattern.MatchString(trimmed) {
		return errors.New("invalid database name: only letters, numbers and underscore are allowed")
	}
	if _, blocked := reservedDBNames[strings.ToLower(trimmed)]; blocked {
		return fmt.Errorf("database name %q is reserved", trimmed)
	}
	return nil
}

// ResolveDBName returns the selected database name based on explicit flag or DSN.
func (s *DbAdminService) ResolveDBName(explicit string) (string, error) {
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		if err := ValidateDBName(explicit); err != nil {
			return "", err
		}
		return explicit, nil
	}
	name, err := databaseNameFromDSN(s.driver, s.dsn)
	if err != nil {
		return "", err
	}
	if err := ValidateDBName(name); err != nil {
		return "", err
	}
	return name, nil
}

func (s *DbAdminService) sqlDriverName() (string, error) {
	switch s.driver {
	case "postgres":
		return "pgx", nil
	case "mysql":
		return "mysql", nil
	case "sqlserver":
		return "sqlserver", nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", s.driver)
	}
}

func (s *DbAdminService) adminDSN() (string, error) {
	switch s.driver {
	case "postgres":
		parsed, err := url.Parse(s.dsn)
		if err != nil {
			return "", fmt.Errorf("invalid DSN: %w", err)
		}
		parsed.Path = "/postgres"
		return parsed.String(), nil
	case "mysql":
		name, err := databaseNameFromDSN(s.driver, s.dsn)
		if err != nil {
			return "", err
		}
		return strings.Replace(s.dsn, "/"+name, "/information_schema", 1), nil
	case "sqlserver":
		parsed, err := url.Parse(s.dsn)
		if err != nil {
			return "", fmt.Errorf("invalid DSN: %w", err)
		}
		q := parsed.Query()
		q.Set("database", "master")
		parsed.RawQuery = q.Encode()
		return parsed.String(), nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", s.driver)
	}
}

func (s *DbAdminService) databaseDSN(dbName string) (string, error) {
	switch s.driver {
	case "postgres", "mysql", "mongodb":
		parsed, err := url.Parse(s.dsn)
		if err != nil {
			return "", fmt.Errorf("invalid DSN: %w", err)
		}
		parsed.Path = "/" + dbName
		return parsed.String(), nil
	case "sqlserver":
		parsed, err := url.Parse(s.dsn)
		if err != nil {
			return "", fmt.Errorf("invalid DSN: %w", err)
		}
		q := parsed.Query()
		q.Set("database", dbName)
		parsed.RawQuery = q.Encode()
		return parsed.String(), nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", s.driver)
	}
}

func (s *DbAdminService) openAdminDB(ctx context.Context) (*sql.DB, error) {
	driverName, err := s.sqlDriverName()
	if err != nil {
		return nil, err
	}
	adminDSN, err := s.adminDSN()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(driverName, adminDSN)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// Exists checks if a database already exists.
func (s *DbAdminService) Exists(ctx context.Context, dbName string) (bool, error) {
	if err := ValidateDBName(dbName); err != nil {
		return false, err
	}

	if s.driver == "mongodb" {
		result, err := s.mongoEval(ctx, fmt.Sprintf("db.getMongo().getDBNames().includes('%s')", dbName))
		if err != nil {
			return false, err
		}
		return strings.TrimSpace(result) == "true", nil
	}

	db, err := s.openAdminDB(ctx)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var query string
	switch s.driver {
	case "postgres":
		query = "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	case "mysql":
		query = "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = ?)"
	case "sqlserver":
		query = "SELECT CASE WHEN EXISTS(SELECT 1 FROM sys.databases WHERE name = @p1) THEN 1 ELSE 0 END"
	default:
		return false, fmt.Errorf("unsupported driver: %s", s.driver)
	}

	var exists bool
	if err := db.QueryRowContext(ctx, query, dbName).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// Create creates a database only when it doesn't exist.
func (s *DbAdminService) Create(ctx context.Context, dbName string) (bool, error) {
	exists, err := s.Exists(ctx, dbName)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	if s.driver == "mongodb" {
		_, err := s.mongoEval(ctx, fmt.Sprintf("db.getSiblingDB('%s').createCollection('driftflow_init')", dbName))
		if err != nil {
			return false, err
		}
		return true, nil
	}

	db, err := s.openAdminDB(ctx)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var stmt string
	switch s.driver {
	case "postgres":
		stmt = fmt.Sprintf("CREATE DATABASE %s", quotePostgresIdent(dbName))
	case "mysql":
		stmt = fmt.Sprintf("CREATE DATABASE %s", quoteMySQLIdent(dbName))
	case "sqlserver":
		stmt = fmt.Sprintf("CREATE DATABASE %s", quoteMSSQLIdent(dbName))
	default:
		return false, fmt.Errorf("unsupported driver: %s", s.driver)
	}

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return false, err
	}
	return true, nil
}

// Drop deletes a database if it exists.
func (s *DbAdminService) Drop(ctx context.Context, dbName string) (bool, error) {
	if err := ValidateDBName(dbName); err != nil {
		return false, err
	}
	exists, err := s.Exists(ctx, dbName)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	if s.driver == "mongodb" {
		_, err := s.mongoEval(ctx, fmt.Sprintf("db.getSiblingDB('%s').dropDatabase()", dbName))
		if err != nil {
			return false, err
		}
		return true, nil
	}

	db, err := s.openAdminDB(ctx)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var stmt string
	switch s.driver {
	case "postgres":
		stmt = fmt.Sprintf("DROP DATABASE %s", quotePostgresIdent(dbName))
	case "mysql":
		stmt = fmt.Sprintf("DROP DATABASE %s", quoteMySQLIdent(dbName))
	case "sqlserver":
		stmt = fmt.Sprintf("DROP DATABASE %s", quoteMSSQLIdent(dbName))
	default:
		return false, fmt.Errorf("unsupported driver: %s", s.driver)
	}

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return false, err
	}
	return true, nil
}

// Backup creates a full dump into output.
func (s *DbAdminService) Backup(ctx context.Context, dbName, output string) error {
	if err := ValidateDBName(dbName); err != nil {
		return err
	}
	if strings.TrimSpace(output) == "" {
		return errors.New("output is required")
	}
	output = filepath.Clean(output)

	switch s.driver {
	case "postgres":
		return s.backupPostgres(ctx, dbName, output)
	case "mysql":
		return s.backupMySQL(ctx, dbName, output)
	case "sqlserver":
		return s.backupSQLServer(ctx, dbName, output)
	case "mongodb":
		return s.backupMongo(ctx, dbName, output)
	default:
		return fmt.Errorf("unsupported driver: %s", s.driver)
	}
}

// Restore restores database content from a backup file.
func (s *DbAdminService) Restore(ctx context.Context, dbName, file string) error {
	if err := ValidateDBName(dbName); err != nil {
		return err
	}
	if strings.TrimSpace(file) == "" {
		return errors.New("file is required")
	}
	if _, err := os.Stat(file); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	switch s.driver {
	case "postgres":
		return s.restorePostgres(ctx, dbName, file)
	case "mysql":
		return s.restoreMySQL(ctx, dbName, file)
	case "sqlserver":
		return s.restoreSQLServer(ctx, dbName, file)
	case "mongodb":
		return s.restoreMongo(ctx, dbName, file)
	default:
		return fmt.Errorf("unsupported driver: %s", s.driver)
	}
}

func (s *DbAdminService) backupPostgres(ctx context.Context, dbName, output string) error {
	dsn, err := s.databaseDSN(dbName)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "pg_dump", "--no-password", "--dbname", dsn, "--file", output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), buildPgPasswordEnv(s.dsn)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) restorePostgres(ctx context.Context, dbName, file string) error {
	dsn, err := s.databaseDSN(dbName)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "psql", "--no-password", "--dbname", dsn, "--file", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), buildPgPasswordEnv(s.dsn)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) backupMySQL(ctx context.Context, dbName, output string) error {
	args, env, err := mysqlClientArgs(s.dsn, dbName)
	if err != nil {
		return err
	}
	args = append(args, "--result-file", output, dbName)
	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) restoreMySQL(ctx context.Context, dbName, file string) error {
	args, env, err := mysqlClientArgs(s.dsn, dbName)
	if err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	cmd := exec.CommandContext(ctx, "mysql", args...)
	cmd.Stdin = f
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql restore failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) backupSQLServer(ctx context.Context, dbName, output string) error {
	db, err := s.openAdminDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	stmt := fmt.Sprintf("BACKUP DATABASE %s TO DISK = @p1 WITH INIT", quoteMSSQLIdent(dbName))
	if _, err := db.ExecContext(ctx, stmt, output); err != nil {
		return fmt.Errorf("sqlserver backup failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) restoreSQLServer(ctx context.Context, dbName, file string) error {
	db, err := s.openAdminDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s SET SINGLE_USER WITH ROLLBACK IMMEDIATE", quoteMSSQLIdent(dbName))); err != nil {
		return fmt.Errorf("sqlserver restore preparation failed: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("RESTORE DATABASE %s FROM DISK = @p1 WITH REPLACE", quoteMSSQLIdent(dbName)), file); err != nil {
		return fmt.Errorf("sqlserver restore failed: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER DATABASE %s SET MULTI_USER", quoteMSSQLIdent(dbName))); err != nil {
		return fmt.Errorf("sqlserver restore finalize failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) backupMongo(ctx context.Context, dbName, output string) error {
	cmd := exec.CommandContext(ctx, "mongodump", "--uri", s.dsn, "--db", dbName, "--archive", output)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongodump failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) restoreMongo(ctx context.Context, dbName, file string) error {
	targetDSN, err := s.databaseDSN(dbName)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "mongorestore", "--uri", targetDSN, "--db", dbName, "--drop", "--archive", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongorestore failed: %w", err)
	}
	return nil
}

func (s *DbAdminService) mongoEval(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "mongosh", "--quiet", s.dsn, "--eval", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mongosh failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func databaseNameFromDSN(driver, dsn string) (string, error) {
	switch strings.ToLower(driver) {
	case "postgres", "mysql", "mongodb":
		parsed, err := url.Parse(dsn)
		if err != nil {
			return "", fmt.Errorf("invalid DSN: %w", err)
		}
		name := strings.TrimPrefix(parsed.Path, "/")
		if name == "" {
			return "", errors.New("database name is missing from DSN")
		}
		return name, nil
	case "sqlserver":
		parsed, err := url.Parse(dsn)
		if err != nil {
			return "", fmt.Errorf("invalid DSN: %w", err)
		}
		name := parsed.Query().Get("database")
		if name == "" {
			return "", errors.New("database name is missing from DSN")
		}
		return name, nil
	default:
		return "", fmt.Errorf("unsupported driver: %s", driver)
	}
}

func mysqlClientArgs(dsn, dbName string) ([]string, []string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid DSN: %w", err)
	}
	host := parsed.Host
	if strings.HasPrefix(host, "tcp(") && strings.HasSuffix(host, ")") {
		host = strings.TrimSuffix(strings.TrimPrefix(host, "tcp("), ")")
	}
	h, p, err := net.SplitHostPort(host)
	if err != nil {
		h = host
		p = "3306"
	}
	args := []string{"--host", h, "--port", p, "--user", parsed.User.Username(), dbName}
	env := []string{}
	if password, ok := parsed.User.Password(); ok && password != "" {
		env = append(env, "MYSQL_PWD="+password)
	}
	return args, env, nil
}

func buildPgPasswordEnv(dsn string) []string {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.User == nil {
		return nil
	}
	password, ok := parsed.User.Password()
	if !ok || password == "" {
		return nil
	}
	return []string{"PGPASSWORD=" + password}
}
