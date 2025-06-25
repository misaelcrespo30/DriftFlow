package driftflow

import (
	"fmt"

	"github.com/misaelcrespo30/DriftFlow/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// ConnectToDB opens a database connection using the given DSN and driver. If
// either parameter is empty, configuration is loaded from environment variables
// (including values from a .env file if present).
func ConnectToDB(dsn string, driver string) (*gorm.DB, error) {
	if dsn == "" || driver == "" {
		cfg := config.Load()
		if dsn == "" {
			dsn = cfg.DSN
		}
		if driver == "" {
			driver = cfg.Driver
		}
	}

	switch driver {
	case "postgres":
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "mysql":
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case "sqlserver":
		return gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}
