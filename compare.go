package driftflow

import (
	"fmt"

	"gorm.io/gorm"
)

// CompareDBs returns human readable differences between two database schemas.
func CompareDBs(from, to *gorm.DB) ([]string, error) {
	fromSchema, err := schemaMap(from)
	if err != nil {
		return nil, err
	}
	toSchema, err := schemaMap(to)
	if err != nil {
		return nil, err
	}
	return diffSchemas(fromSchema, toSchema), nil
}

type tableInfo map[string]string

type schemaInfo map[string]tableInfo

func schemaMap(db *gorm.DB) (schemaInfo, error) {
	tables, err := db.Migrator().GetTables()
	if err != nil {
		return nil, err
	}
	s := make(schemaInfo)
	for _, t := range tables {
		cols, err := db.Migrator().ColumnTypes(t)
		if err != nil {
			return nil, err
		}
		colMap := make(tableInfo)
		for _, c := range cols {
			colMap[c.Name()] = c.DatabaseTypeName()
		}
		s[t] = colMap
	}
	return s, nil
}

func diffSchemas(from, to schemaInfo) []string {
	var diffs []string
	for table := range from {
		if _, ok := to[table]; !ok {
			diffs = append(diffs, fmt.Sprintf("[-] table %s", table))
		}
	}
	for table := range to {
		if _, ok := from[table]; !ok {
			diffs = append(diffs, fmt.Sprintf("[+] table %s", table))
		}
	}
	for table, fromCols := range from {
		toCols, ok := to[table]
		if !ok {
			continue
		}
		for col := range fromCols {
			if _, ok := toCols[col]; !ok {
				diffs = append(diffs, fmt.Sprintf("[-] column %s.%s", table, col))
			}
		}
		for col := range toCols {
			if _, ok := fromCols[col]; !ok {
				diffs = append(diffs, fmt.Sprintf("[+] column %s.%s", table, col))
			}
		}
		for col, ft := range fromCols {
			if tt, ok := toCols[col]; ok && ft != tt {
				diffs = append(diffs, fmt.Sprintf("[~] column %s.%s %s -> %s", table, col, ft, tt))
			}
		}
	}
	return diffs
}
