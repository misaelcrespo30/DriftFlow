package driftflow

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type IndexDefinition struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Where   string   `json:"where,omitempty"`
}

type indexKind string

const (
	indexKindIndex            indexKind = "index"
	indexKindUniqueIndex      indexKind = "unique_index"
	indexKindUniqueConstraint indexKind = "unique_constraint"
)

type indexTag struct {
	Name     string
	Priority int
	Unique   bool
	Kind     indexKind
}

type indexColumn struct {
	Column   string
	Priority int
	Order    int
}

type indexPlan struct {
	Name    string
	Unique  bool
	Columns []indexColumn
}

func isGormDeletedAtType(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct && t.PkgPath() == "gorm.io/gorm" && t.Name() == "DeletedAt"
}

func modelDeletedAtInfo(t reflect.Type) (bool, string) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false, ""
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() || f.Tag.Get("gorm") == "-" {
			continue
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if f.Anonymous && ft.Kind() == reflect.Struct {
			if ok, col := modelDeletedAtInfo(ft); ok {
				return true, col
			}
		}
		if isGormDeletedAtType(ft) {
			name := getTagValue(f.Tag.Get("gorm"), "column")
			if name == "" {
				name = toSnakeCase(f.Name)
			}
			return true, name
		}
	}
	return false, ""
}

func parseIndexTags(tag string) []indexTag {
	if tag == "" {
		return nil
	}
	parts := strings.Split(tag, ";")
	var tags []indexTag
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		switch {
		case lower == "unique":
			tags = append(tags, indexTag{Unique: true, Kind: indexKindUniqueConstraint})
		case strings.HasPrefix(lower, "uniqueindex"):
			name, priority, _ := parseIndexClause(part)
			tags = append(tags, indexTag{Name: name, Priority: priority, Unique: true, Kind: indexKindUniqueIndex})
		case strings.HasPrefix(lower, "index"):
			name, priority, unique := parseIndexClause(part)
			tags = append(tags, indexTag{Name: name, Priority: priority, Unique: unique, Kind: indexKindIndex})
		}
	}
	return tags
}

func parseIndexClause(part string) (string, int, bool) {
	name := ""
	priority := 0
	unique := false
	segments := strings.SplitN(part, ":", 2)
	if len(segments) == 2 {
		opts := strings.Split(segments[1], ",")
		for _, opt := range opts {
			opt = strings.TrimSpace(opt)
			if opt == "" {
				continue
			}
			lower := strings.ToLower(opt)
			switch {
			case strings.HasPrefix(lower, "priority"):
				kv := strings.SplitN(opt, ":", 2)
				if len(kv) == 2 {
					if val, err := strconv.Atoi(strings.TrimSpace(kv[1])); err == nil {
						priority = val
					}
				}
			case lower == "unique":
				unique = true
			default:
				if name == "" {
					name = opt
				}
			}
		}
	}
	return name, priority, unique
}

func addIndexPlan(plans map[string]*indexPlan, tag indexTag, column string, order int) {
	name := strings.TrimSpace(tag.Name)
	if name == "" {
		name = fmt.Sprintf("__implicit_%s_%d", column, order)
	}
	plan, ok := plans[name]
	if !ok {
		plan = &indexPlan{Name: name}
		plans[name] = plan
	}
	if tag.Unique {
		plan.Unique = true
	}
	plan.Columns = append(plan.Columns, indexColumn{Column: column, Priority: tag.Priority, Order: order})
}

func buildIndexDefinitions(table string, plans map[string]*indexPlan, deletedAtCol string, engine string, hasSoftDelete bool) []IndexDefinition {
	if len(plans) == 0 {
		return nil
	}
	engine = normalizeEngine(engine)
	var indexes []IndexDefinition
	for name, plan := range plans {
		cols := make([]indexColumn, len(plan.Columns))
		copy(cols, plan.Columns)
		sort.SliceStable(cols, func(i, j int) bool {
			if cols[i].Priority > 0 || cols[j].Priority > 0 {
				if cols[i].Priority == 0 {
					return false
				}
				if cols[j].Priority == 0 {
					return true
				}
				if cols[i].Priority != cols[j].Priority {
					return cols[i].Priority < cols[j].Priority
				}
			}
			return cols[i].Order < cols[j].Order
		})
		var columns []string
		for _, col := range cols {
			columns = append(columns, col.Column)
		}
		partial := engine == "postgres" && hasSoftDelete && deletedAtCol != "" && !containsColumn(columns, deletedAtCol)
		idxName := name
		if strings.HasPrefix(idxName, "__implicit_") {
			idxName = defaultIndexName(table, columns, plan.Unique, partial)
		} else if partial {
			idxName = ensureActiveSuffix(idxName)
		}

		var where string
		if partial {
			where = fmt.Sprintf("%s IS NULL", quoteIdent(engine, deletedAtCol))
		}

		indexes = append(indexes, IndexDefinition{
			Name:    idxName,
			Columns: columns,
			Unique:  plan.Unique,
			Where:   where,
		})
	}

	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].Name < indexes[j].Name
	})
	return indexes
}

func defaultIndexName(table string, columns []string, unique bool, partial bool) string {
	prefix := "ix"
	if unique {
		prefix = "ux"
	}
	base := fmt.Sprintf("%s_%s_%s", prefix, table, strings.Join(columns, "_"))
	if partial {
		base += "_active"
	}
	return base
}

func ensureActiveSuffix(name string) string {
	if strings.HasSuffix(name, "_active") {
		return name
	}
	return name + "_active"
}

func containsColumn(columns []string, target string) bool {
	for _, col := range columns {
		if col == target {
			return true
		}
	}
	return false
}

func cloneIndexes(indexes []IndexDefinition) []IndexDefinition {
	if len(indexes) == 0 {
		return nil
	}
	out := make([]IndexDefinition, 0, len(indexes))
	for _, idx := range indexes {
		cols := append([]string{}, idx.Columns...)
		out = append(out, IndexDefinition{
			Name:    idx.Name,
			Columns: cols,
			Unique:  idx.Unique,
			Where:   idx.Where,
		})
	}
	return out
}
