//go:build atlas

package schemaflow

import (
	"context"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
	"gorm.io/gorm"

	"matters-service/schemaflow/internal/diff"
	"matters-service/schemaflow/internal/executor"
	"matters-service/schemaflow/internal/generator"
	"matters-service/schemaflow/internal/loader"
	"matters-service/schemaflow/internal/state"
)

// Diff prints schema differences between the current database and the given models.
func Diff(ctx context.Context, gdb *gorm.DB, client *sqlclient.Client, models []interface{}) error {
	return diff.Run(ctx, gdb, client, models)
}

// Generate creates migration files for changes detected in the given models.
func Generate(ctx context.Context, models []interface{}, gdb *gorm.DB, client *sqlclient.Client, dir migrate.Dir) error {
	return generator.Generate(ctx, models, gdb, client, dir)
}

// Up applies all pending migrations in the provided directory.
func Up(ctx context.Context, client *sqlclient.Client, dir migrate.Dir) error {
	return executor.Up(ctx, client, dir)
}

// Down rolls back migrations until the provided target is reached.
func Down(ctx context.Context, client *sqlclient.Client, dir migrate.Dir, target string) error {
	return executor.Down(ctx, client, dir, target)
}

// LoadState reads the migration state from the given directory.
func LoadState(ctx context.Context, path string) (*state.State, error) {
	return loader.Load(ctx, path)
}
