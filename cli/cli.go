package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"

	driftflow "github.com/misaelcrespo30/DriftFlow"
	"github.com/misaelcrespo30/DriftFlow/config"
	"github.com/misaelcrespo30/DriftFlow/helpers"
)

var (
	dsn        string
	driver     string
	migDir     string
	seedGenDir string
	seedRunDir string
	modelsDir  string
)

// NewRootCommand builds the DriftFlow CLI root command. It can be used by
// external applications to run DriftFlow commands programmatically.
func NewRootCommand() *cobra.Command {
	cfg := config.Load()
	dsn = cfg.DSN
	driver = cfg.Driver
	migDir = cfg.MigDir
	seedGenDir = cfg.SeedGenDir
	seedRunDir = cfg.SeedRunDir

	rootCmd := &cobra.Command{Use: "driftflow"}
	rootCmd.PersistentFlags().StringVar(&dsn, "dsn", cfg.DSN, "database DSN")
	rootCmd.PersistentFlags().StringVar(&driver, "driver", cfg.Driver, "database driver")
	rootCmd.PersistentFlags().StringVar(&migDir, "migrations", cfg.MigDir, "migrations directory")
	//rootCmd.PersistentFlags().StringVar(&seedGenDir, "seeds", cfg.SeedGenDir, "seed data directory")
	//rootCmd.PersistentFlags().StringVar(&seedRunDir, "seeds", cfg.SeedRunDir, "seed data directory")
	rootCmd.PersistentFlags().StringVar(&seedRunDir, "seeds", cfg.SeedRunDir, "seed run directory")
	rootCmd.PersistentFlags().StringVar(&seedGenDir, "seed-gen-dir", cfg.SeedGenDir, "seed generation directory")
	rootCmd.PersistentFlags().StringVar(&modelsDir, "models", cfg.ModelsDir, "models directory")

	rootCmd.AddCommand(Commands(cfg)...)

	return rootCmd
}

// Execute runs the DriftFlow CLI using the default configuration.
func Execute() error {
	return NewRootCommand().Execute()
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
	return nil, fmt.Errorf("unsupported DSN: %s", d)
}

func newUpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			return driftflow.Up(db, migDir)
		},
	}
}

func newDownCommand() *cobra.Command {
	return &cobra.Command{
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
	}
}

func newUndoCommand() *cobra.Command {
	return &cobra.Command{
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
	}
}

func newRollbackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback [n]",
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
	}
}

func newSeedCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Ejecuta los seeders del proyecto registrado",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			return driftflow.Seed(db, seedRunDir)

		},
	}
}

func newSeedgenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seedgen",
		Short: "Generate JSON seed templates from models",
		RunE: func(cmd *cobra.Command, args []string) error {
			models, err := helpers.LoadModels()
			if err != nil {
				return err
			}
			return driftflow.GenerateSeedAssets(models, seedGenDir)
		},
	}
	return cmd
}

/*
	func newGenerateCommand() *cobra.Command {
		return &cobra.Command{
			Use:   "generate",
			Short: "Generate migration files from models (snapshot + incremental)",
			RunE: func(cmd *cobra.Command, args []string) error {
				models, err := helpers.LoadModels()
				if err != nil {
					return err
				}
				return driftflow.GenerateModelMigrations(models, migDir)
			},
		}
	}
*/

func newGenerateCommand() *cobra.Command {
	var repair bool
	var adopt bool

	return &cobra.Command{
		Use:   "generate",
		Short: "Generate migration files from models (snapshot + incremental)",
		RunE: func(cmd *cobra.Command, args []string) error {
			models, err := helpers.LoadModels()
			if err != nil {
				return err
			}

			opts := driftflow.GenerateOptions{
				Dir:          migDir,
				ManifestMode: driftflow.ManifestStrict, // default
				Engine:       driver,
			}

			if repair {
				opts.ManifestMode = driftflow.ManifestRepair
				opts.RepairAddUntracked = adopt
			}

			return driftflow.GenerateModelMigrations(models, opts)
		},
	}

	/*cmd.Flags().BoolVar(&repair, "repair", false, "Repair modified migration files (recalculate hashes)")
	cmd.Flags().BoolVar(&adopt, "adopt", false, "Adopt untracked migration files into manifest (requires --repair)")

	return cmd*/
}

func newMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Generate and apply migrations from models",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			models, err := helpers.LoadModels()
			if err != nil {
				return err
			}
			return driftflow.Migrate(db, migDir, models)
		},
	}
}

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate migration files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return driftflow.Validate(migDir)
		},
	}
}

func newResetCommand() *cobra.Command {
	var force bool
	var allowProd bool
	var schema string
	var database string

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Drop all tables in the target database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !allowProd && strings.EqualFold(os.Getenv("ENV"), "production") {
				return fmt.Errorf("reset blocked in production; use --allow-prod to override")
			}
			if !force {
				reader := bufio.NewReader(os.Stdin)
				fmt.Fprint(cmd.OutOrStdout(), "This will DROP ALL TABLES in the target database. Type 'yes' to continue: ")
				input, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return err
				}
				if strings.TrimSpace(input) != "yes" {
					return fmt.Errorf("reset aborted")
				}
			}
			db, err := openDB()
			if err != nil {
				return err
			}
			summary, err := driftflow.Reset(db, driftflow.ResetOptions{
				DSN:      dsn,
				Driver:   driver,
				Schema:   schema,
				Database: database,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Dialect: %s\n", summary.Dialect)
			fmt.Fprintf(cmd.OutOrStdout(), "Database: %s\n", summary.Database)
			fmt.Fprintf(cmd.OutOrStdout(), "Schema: %s\n", summary.Schema)
			fmt.Fprintf(cmd.OutOrStdout(), "Tables dropped: %d\n", summary.TablesDropped)
			fmt.Fprintf(cmd.OutOrStdout(), "Recreated schema: %t\n", summary.RecreatedSchema)
			fmt.Fprintf(cmd.OutOrStdout(), "Recreated database: %t\n", summary.RecreatedDatabase)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&allowProd, "allow-prod", false, "allow reset when ENV=production")
	cmd.Flags().StringVar(&schema, "schema", "", "database schema to reset")
	cmd.Flags().StringVar(&database, "database", "", "database name to reset")
	return cmd
}

func newCleanCommand() *cobra.Command {
	var force bool
	var allowProd bool
	var schema string
	var include string
	var exclude string
	var keepMigrations bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "clean",
		Aliases: []string{"truncate"},
		Short:   "Delete all data in tables without dropping the schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !allowProd && strings.EqualFold(os.Getenv("ENV"), "production") {
				return fmt.Errorf("clean blocked in production; use --allow-prod to override")
			}
			if !force {
				reader := bufio.NewReader(os.Stdin)
				fmt.Fprint(cmd.OutOrStdout(), "This will DELETE/TRUNCATE all data in all tables but keep the schema. Type 'yes' to continue: ")
				input, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return err
				}
				if strings.TrimSpace(input) != "yes" {
					return fmt.Errorf("clean aborted")
				}
			}

			db, err := openDB()
			if err != nil {
				return err
			}

			summary, err := driftflow.Clean(db, driftflow.CleanOptions{
				DSN:            dsn,
				Driver:         driver,
				Schema:         schema,
				IncludePattern: include,
				ExcludePattern: exclude,
				KeepMigrations: keepMigrations,
				DryRun:         dryRun,
			})
			if err != nil {
				return err
			}

			if dryRun {
				for _, stmt := range summary.Statements {
					fmt.Fprintln(cmd.OutOrStdout(), stmt)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Dialect: %s\n", summary.Dialect)
			fmt.Fprintf(cmd.OutOrStdout(), "Schema: %s\n", summary.Schema)
			fmt.Fprintf(cmd.OutOrStdout(), "Tables affected: %d\n", summary.TablesAffected)
			fmt.Fprintf(cmd.OutOrStdout(), "Method: %s\n", summary.Method)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&allowProd, "allow-prod", false, "allow clean when ENV=production")
	cmd.Flags().StringVar(&schema, "schema", "", "database schema to clean")
	cmd.Flags().StringVar(&include, "include", "", "include only tables matching the pattern")
	cmd.Flags().StringVar(&exclude, "exclude", "", "exclude tables matching the pattern")
	cmd.Flags().BoolVar(&keepMigrations, "keep-migrations", true, "keep migration/history tables")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print statements without executing")
	return cmd
}

func newAuditCommand() *cobra.Command {
	auditCmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit log commands",
	}
	auditCmd.AddCommand(newAuditListCommand())
	auditCmd.AddCommand(newAuditExportCommand())
	return auditCmd
}

func newAuditListCommand() *cobra.Command {
	return &cobra.Command{
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
			//  Manejo de error en encabezado
			if _, err := fmt.Fprintln(w, "ID\tVERSION\tCOMMIT\tACTION\tUSER\tHOST\tLOGGED_AT"); err != nil {
				return fmt.Errorf("error escribiendo encabezado: %w", err)
			}
			for _, l := range logs {
				if _, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
					l.ID, l.Version, l.Commit, l.Action, l.User, l.Host, l.LoggedAt.Format(time.RFC3339),
				); err != nil {
					return fmt.Errorf("error escribiendo fila de log ID %d: %w", l.ID, err)
				}
			}
			// Manejo de error en Flush
			if err := w.Flush(); err != nil {
				return fmt.Errorf("error al vaciar salida del tabwriter: %w", err)
			}
			return nil
		},
	}
}

func newAuditExportCommand() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export audit log",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return fmt.Errorf("error al abrir la conexión a la base de datos: %w", err)
			}

			logs, err := driftflow.ListAuditLog(db)
			if err != nil {
				return fmt.Errorf("error al obtener logs de auditoría: %w", err)
			}

			if jsonOut {
				b, err := json.MarshalIndent(logs, "", "  ")
				if err != nil {
					return fmt.Errorf("error al serializar logs a JSON: %w", err)
				}
				_, err = fmt.Println(string(b))
				if err != nil {
					return fmt.Errorf("error al imprimir salida JSON: %w", err)
				}
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			if _, err := fmt.Fprintln(w, "ID\tVERSION\tCOMMIT\tACTION\tUSER\tHOST\tLOGGED_AT"); err != nil {
				return fmt.Errorf("error al escribir encabezado: %w", err)
			}

			for _, l := range logs {
				if _, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
					l.ID, l.Version, l.Commit, l.Action, l.User, l.Host, l.LoggedAt.Format(time.RFC3339)); err != nil {
					return fmt.Errorf("error al escribir fila de log ID %d: %w", l.ID, err)
				}
			}

			if err := w.Flush(); err != nil {
				return fmt.Errorf("error al vaciar salida del tabwriter: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "output as JSON")
	return cmd
}

func newCompareCommand() *cobra.Command {
	var fromDSN, toDSN string
	cmd := &cobra.Command{
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
				case strings.HasPrefix(d, "[+"):
					fmt.Printf("\033[32m%s\033[0m\n", d)
				case strings.HasPrefix(d, "[-"):
					fmt.Printf("\033[31m%s\033[0m\n", d)
				case strings.HasPrefix(d, "[~"):
					fmt.Printf("\033[33m%s\033[0m\n", d)
				default:
					fmt.Println(d)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&fromDSN, "from", "", "source DSN")
	cmd.Flags().StringVar(&toDSN, "to", "", "target DSN")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

/////////////////////////////////////

func Commands(cfg *config.Config) []*cobra.Command {
	// Sobrescribe las variables globales para compatibilidad con funciones actuales
	dsn = cfg.DSN
	driver = cfg.Driver
	migDir = cfg.MigDir
	seedGenDir = cfg.SeedGenDir
	seedRunDir = cfg.SeedRunDir
	modelsDir = cfg.ModelsDir

	return []*cobra.Command{
		newUpCommand(),
		newDownCommand(),
		newUndoCommand(),
		newRollbackCommand(),
		newResetCommand(),
		newCleanCommand(),
		newSeedCommand(),
		newSeedgenCommand(),
		newGenerateCommand(),
		newMigrateCommand(),
		newValidateCommand(),
		newAuditCommand(),
		newCompareCommand(),
	}
}
