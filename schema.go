package driftflow

import "gorm.io/gorm"

// ExtractSchema returns a map describing the database schema.
// The returned map is keyed by table name, then column name with
// the associated database type as value.
func ExtractSchema(db *gorm.DB) (map[string]map[string]string, error) {
	tables, err := db.Migrator().GetTables()
	if err != nil {
		return nil, err
	}
	schema := make(map[string]map[string]string, len(tables))
	for _, tbl := range tables {
		cols, err := db.Migrator().ColumnTypes(tbl)
		if err != nil {
			return nil, err
		}
		cmap := make(map[string]string, len(cols))
		for _, c := range cols {
			name := c.Name()
			typ := c.DatabaseTypeName()
			cmap[name] = typ
		}
		schema[tbl] = cmap
	}
	return schema, nil
}
