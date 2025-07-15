package config

import (
	"os"
	"strings"
)

var defaultEnv = map[string]string{
	"DB_TYPE":      "postgres",
	"DSN":          "",
	"DB_HOST":      "localhost",
	"DB_PORT":      "5432",
	"DB_NAME":      "driftflow",
	"DB_USER":      "user",
	"DB_PASSWORD":  "password",
	"DB_SSLMODE":   "disable",
	"MIG_DIR":      "internal/database/migrations",
	"SEED_GEN_DIR": "internal/database/data",
	"SEED_RUN_DIR": "internal/database/seed",
	"MODELS_DIR":   "internal/models",
	"PROJECT_PATH": "",
}

var defaultEnvOrder = []string{
	"DB_TYPE",
	"DSN",
	"DB_HOST",
	"DB_PORT",
	"DB_NAME",
	"DB_USER",
	"DB_PASSWORD",
	"DB_SSLMODE",
	"MIG_DIR",
	"SEED_GEN_DIR",
	"SEED_RUN_DIR",
	"MODELS_DIR",
	"PROJECT_PATH",
}

var defaultEnvContent = buildDefaultEnvContent()

func buildDefaultEnvContent() string {
	var sb strings.Builder
	for _, k := range defaultEnvOrder {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(defaultEnv[k])
		sb.WriteByte('\n')
	}
	return sb.String()
}

// EnsureEnvFile creates a .env file at path if it doesn't exist using the
// default environment values.
func EnsureEnvFile(path string) error {
	if path == "" {
		path = ".env"
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(defaultEnvContent), 0o644)
}
