package loader

import (
	"context"
	"path/filepath"

	"DriftFlow/config"
	"DriftFlow/internal/state"
)

// Load returns the migration state by reading `.sql` files from dir. If dir is
// empty, it falls back to the `MIG_DIR` configuration loaded from the
// environment.
func Load(_ context.Context, dir string) (*state.State, error) {
	if dir == "" {
		dir = config.Load().MigDir
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	return &state.State{Files: files}, nil
}
