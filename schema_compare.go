package driftflow

import (
	"fmt"
	"sort"
)

// CompareSchemas compares two database schema representations.
// The schema maps table names to a map of column names and their types.
// It returns a list of differences such as missing tables, columns or type changes.
func CompareSchemas(source map[string]map[string]string, target map[string]map[string]string) []string {
	diffs := []string{}

	// Check tables from source against target
	for tbl, srcCols := range source {
		tgtCols, ok := target[tbl]
		if !ok {
			diffs = append(diffs, fmt.Sprintf("missing table: %s", tbl))
			continue
		}

		// Columns present in source but missing or changed in target
		for col, srcType := range srcCols {
			tgtType, ok := tgtCols[col]
			if !ok {
				diffs = append(diffs, fmt.Sprintf("missing column: %s.%s", tbl, col))
				continue
			}
			if srcType != tgtType {
				diffs = append(diffs, fmt.Sprintf("type mismatch for %s.%s: %s vs %s", tbl, col, srcType, tgtType))
			}
		}

		// Extra columns in target that are not in source
		for col := range tgtCols {
			if _, ok := srcCols[col]; !ok {
				diffs = append(diffs, fmt.Sprintf("extra column: %s.%s", tbl, col))
			}
		}
	}

	// Extra tables in target not present in source
	for tbl := range target {
		if _, ok := source[tbl]; !ok {
			diffs = append(diffs, fmt.Sprintf("extra table: %s", tbl))
		}
	}

	sort.Strings(diffs)
	return diffs
}
