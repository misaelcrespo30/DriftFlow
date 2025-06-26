package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains the minimal configuration required by the DriftFlow CLI.
type Config struct {
	DSN     string
	Driver  string
	MigDir  string
	SeedDir string
}

// Load reads environment variables (from the system or a .env file) and
// returns a Config struct. If DSN is not provided directly, it will be
// constructed from standard database variables.
func Load() *Config {
	_ = loadEnvFile()

	driver := getEnvOrDefault("DB_TYPE", "postgres")
	cfg := &Config{
		DSN:     os.Getenv("DSN"),
		Driver:  driver,
		MigDir:  getEnvOrDefault("MIG_DIR", "migrations"),
		SeedDir: getEnvOrDefault("SEED_DIR", "seeds"),
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
	default:
		return ""
	}
}

// loadEnvFile searches for a .env file in the working directory and its parents.
// If none is found, it falls back to the default .env bundled with the library.
func loadEnvFile() error {
	wd, err := os.Getwd()
	if err == nil {
		dir := wd
		for {
			path := filepath.Join(dir, ".env")
			if _, err := os.Stat(path); err == nil {
				return godotenv.Load(path)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	if _, file, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Join(filepath.Dir(file), "..")
		return godotenv.Load(filepath.Join(repoRoot, ".env"))
	}
	return nil
}

// ValidateDir checks that dir exists and is a directory.
func ValidateDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", dir)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", dir)
	}
	return nil
}

// ValidateDirs verifies that both migration and seed directories exist.
func ValidateDirs(migDir, seedDir string) error {
	if err := ValidateDir(migDir); err != nil {
		return err
	}
	if err := ValidateDir(seedDir); err != nil {
		return err
	}
	return nil
}
