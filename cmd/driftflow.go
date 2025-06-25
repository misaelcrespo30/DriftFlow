package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
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
		Use:   "undo [n]",
		Short: "Rollback the last n migrations (default 1)",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			steps := 1
			if len(args) == 1 {
				var err error
				steps, err = strconv.Atoi(args[0])
				if err != nil {
					return err
				}
			}
			db, err := openDB()
			if err != nil {
				return err
			}
			return driftflow.DownSteps(db, migDir, steps)
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

	auditCmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit log commands",
	}

	auditCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			logs, err := driftflow.ListAuditLog(db)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tVERSION\tCOMMIT\tACTION\tUSER\tHOST\tLOGGED_AT")
			for _, l := range logs {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n", l.ID, l.Version, l.Commit, l.Action, l.User, l.Host, l.LoggedAt.Format(time.RFC3339))
			}
			w.Flush()
			return nil
		},
	})

	var jsonOut bool
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Export audit log",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			logs, err := driftflow.ListAuditLog(db)
			if err != nil {
				return err
			}
			if jsonOut {
				b, err := json.MarshalIndent(logs, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(b))
				return nil
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tVERSION\tCOMMIT\tACTION\tUSER\tHOST\tLOGGED_AT")
			for _, l := range logs {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n", l.ID, l.Version, l.Commit, l.Action, l.User, l.Host, l.LoggedAt.Format(time.RFC3339))
			}
			w.Flush()
			return nil
		},
	}
	exportCmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	auditCmd.AddCommand(exportCmd)

	rootCmd.AddCommand(auditCmd)

	var fromDSN, toDSN string
	compareCmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare schemas of two databases",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbFrom, err := openDSN(fromDSN)
			if err != nil {
				return err
			}
			dbTo, err := openDSN(toDSN)
			if err != nil {
				return err
			}
			diffs, err := driftflow.CompareDBs(dbFrom, dbTo)
			if err != nil {
				return err
			}
			for _, d := range diffs {
				switch {
				case strings.HasPrefix(d, "[+]"):
					fmt.Printf("\033[32m%s\033[0m\n", d)
				case strings.HasPrefix(d, "[-]"):
					fmt.Printf("\033[31m%s\033[0m\n", d)
				case strings.HasPrefix(d, "[~]"):
					fmt.Printf("\033[33m%s\033[0m\n", d)
				default:
					fmt.Println(d)
				}
			}
			return nil
		},
	}
	compareCmd.Flags().StringVar(&fromDSN, "from", "", "source DSN")
	compareCmd.Flags().StringVar(&toDSN, "to", "", "target DSN")
	_ = compareCmd.MarkFlagRequired("from")
	_ = compareCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(compareCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func openDB() (*gorm.DB, error) {
	return driftflow.ConnectToDB(dsn, driver)
}

func openDSN(d string) (*gorm.DB, error) {
	if strings.HasPrefix(d, "postgres://") || strings.HasPrefix(d, "postgresql://") {
		return gorm.Open(postgres.Open(d), &gorm.Config{})
	}
	if strings.HasPrefix(d, "mysql://") {
		return gorm.Open(mysql.Open(d), &gorm.Config{})
	}
	if strings.HasPrefix(d, "sqlserver://") {
		return gorm.Open(sqlserver.Open(d), &gorm.Config{})
	}
	return gorm.Open(sqlite.Open(d), &gorm.Config{})
}
