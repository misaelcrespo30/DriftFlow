//go:build atlas

package generator

import (
	"context"
	"fmt"
	"time"

	"ariga.io/atlas-provider-gorm/gormschema"
	"ariga.io/atlas/sql/migrate"
	"ariga.io/atlas/sql/sqlclient"
	"gorm.io/gorm"
)

// Generate runs the migration generation logic.
func Generate(ctx context.Context, models []interface{}, gdb *gorm.DB, client *sqlclient.Client, dir migrate.Dir) error {
	realm, err := gormschema.New(gdb).Load(&gormschema.LoaderConfig{Models: models})
	if err != nil {
		return fmt.Errorf("load models: %w", err)
	}

	current, err := client.InspectRealm(ctx, nil)
	if err != nil {
		return fmt.Errorf("inspect database: %w", err)
	}

	diff, err := client.RealmDiff(current, realm)
	if err != nil {
		return fmt.Errorf("diff schema: %w", err)
	}
	if len(diff) == 0 {
		fmt.Println("schema up to date")
		return nil
	}

	filename := time.Now().Format("20060102150405") + "_auto"
	planner := migrate.NewPlanner(client.Driver, dir)
	if err := planner.WriteDiff(ctx, filename, realm); err != nil {
		return err
	}
	fmt.Printf("migration written: %s.sql\n", filename)
	return nil
}
