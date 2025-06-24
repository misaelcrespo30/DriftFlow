package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains the minimal configuration required by the DriftFlow CLI.
type Config struct {
	DSN    string
	Driver string
	MigDir string
}

// Load reads environment variables (from the system or a .env file) and
// returns a Config struct. If DSN is not provided directly, it will be
// constructed from standard database variables.
func Load() *Config {
	_ = godotenv.Load()

	driver := getEnvOrDefault("DB_TYPE", "postgres")
	cfg := &Config{
		DSN:    os.Getenv("DSN"),
		Driver: driver,
		MigDir: getEnvOrDefault("MIG_DIR", "migrations"),
	}

	if cfg.DSN == "" {
		cfg.DSN = buildDSN(driver)
	}

	return cfg
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// buildDSN constructs a DSN from common environment variables if DSN is not
// provided directly.
func buildDSN(driver string) string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	switch strings.ToLower(driver) {
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, name, sslmode)
	case "mysql":
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%s)/%s", user, pass, host, port, name)
	case "sqlserver":
		return fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s", user, pass, host, port, name)
	case "sqlite":
		return fmt.Sprintf("file:%s?cache=shared", name)
	default:
		return ""
	}
}
