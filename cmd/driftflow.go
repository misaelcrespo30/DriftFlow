package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	driftflow "DriftFlow"
	"DriftFlow/config"
)

var (
	dsn     string
	driver  string
	migDir  string
	seedDir string
)

func main() {
	cfg := config.Load()
	rootCmd := &cobra.Command{Use: "driftflow"}
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", cfg.DSN, "database DSN")
	rootCmd.PersistentFlags().StringVar(&driver, "driver", cfg.Driver, "database driver")
	rootCmd.PersistentFlags().StringVar(&migDir, "migrations", cfg.MigDir, "migrations directory")
	rootCmd.PersistentFlags().StringVar(&seedDir, "seeds", cfg.SeedDir, "seed data directory")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			return driftflow.Up(db, migDir)
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
			return driftflow.Down(db, migDir, args[0])
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "seed",
		Short: "Execute JSON seed files",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			// no default models; users should populate seeders slice in their build
			var seeders []driftflow.Seeder
			return driftflow.Seed(db, seedDir, seeders)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "seedgen",
		Short: "Generate JSON seed templates from models",
		RunE: func(cmd *cobra.Command, args []string) error {
			// users should provide their models implementing Seeder
			var models []interface{}
			return driftflow.GenerateSeedTemplates(models, seedDir)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "generate",
		Short: "Generate migration files from models",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			var models []interface{}
			return driftflow.GenerateMigrations(db, models, migDir)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Generate and apply migrations from models",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			var models []interface{}
			return driftflow.Migrate(db, migDir, models)
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate migration files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return driftflow.Validate(migDir)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func openDB() (*gorm.DB, error) {
	return driftflow.ConnectToDB(dsn, driver)
}
