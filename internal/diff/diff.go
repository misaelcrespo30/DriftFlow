//go:build atlas

package diff

import (
	"context"
	"fmt"

	"ariga.io/atlas-provider-gorm/gormschema"
	"ariga.io/atlas/sql/sqlclient"
	"gorm.io/gorm"
)

// Run executes a schema diff operation.
func Run(ctx context.Context, gdb *gorm.DB, client *sqlclient.Client, models []interface{}) error {
	realm, err := gormschema.New(gdb).Load(&gormschema.LoaderConfig{
		Models: models,
	})
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

	fmt.Printf("%d schema changes detected\n", len(diff))
	for _, c := range diff {
		fmt.Printf("- %T\n", c)
	}
	return nil
}
