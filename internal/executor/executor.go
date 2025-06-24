//go:build atlas

package executor

import (
	"context"

	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
)

// Up applies pending migrations.
func Up(ctx context.Context, client *sqlclient.Client, dir migrate.Dir) error {
	exec, err := migrate.NewExecutor(client.Driver, dir, migrate.WithMigrationTable("atlas_schema_revisions"))
	if err != nil {
		return err
	}
	return exec.Execute(ctx, migrate.Up)
}

// Down rolls back to the specified migration.
func Down(ctx context.Context, client *sqlclient.Client, dir migrate.Dir, target string) error {
	exec, err := migrate.NewExecutor(client.Driver, dir, migrate.WithMigrationTable("atlas_schema_revisions"))
	if err != nil {
		return err
	}
	return exec.Execute(ctx, migrate.DownTarget(target))
}
