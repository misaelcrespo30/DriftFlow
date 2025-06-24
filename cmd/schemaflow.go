package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	schema "matters-service/schemaflow"
	"matters-service/schemaflow/config"
)

var (
	dsn    string
	driver string
	migDir string
)

func main() {
	cfg := config.Load()
	rootCmd := &cobra.Command{Use: "schemaflow"}
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", cfg.DSN, "database DSN")
	rootCmd.PersistentFlags().StringVar(&driver, "driver", cfg.Driver, "database driver")
	rootCmd.PersistentFlags().StringVar(&migDir, "migrations", cfg.MigDir, "migrations directory")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			return schema.Up(db, migDir)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "down [version]",
		Short: "Rollback migrations after the given version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			return schema.Down(db, migDir, args[0])
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func openDB() (*gorm.DB, error) {
	switch driver {
	case "postgres":
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "sqlite":
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	case "mysql":
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}
}
