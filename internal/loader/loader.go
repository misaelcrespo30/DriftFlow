package loader

import (
	"context"
	"os"
	"sort"
	"strings"

	"matters-service/schemaflow/internal/state"
)

// Load reads migration files from the given directory and returns their state.
func Load(_ context.Context, dir string) (*state.State, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return &state.State{Files: files}, nil
}
