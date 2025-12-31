package driftflow

import "strings"

type ColAlter struct {
	From string
	To   string
}

// diffSnapshot compares previous snapshot state vs current model state.
func diffSnapshot(prev, next map[string]string) (added map[string]string, removed map[string]string, altered map[string]ColAlter) {
	added = map[string]string{}
	removed = map[string]string{}
	altered = map[string]ColAlter{}

	// added + altered
	for col, nextDef := range next {
		prevDef, ok := prev[col]
		if !ok {
			added[col] = nextDef
			continue
		}
		if normalizeDef(prevDef) != normalizeDef(nextDef) {
			altered[col] = ColAlter{From: prevDef, To: nextDef}
		}
	}

	// removed
	for col, prevDef := range prev {
		if _, ok := next[col]; !ok {
			removed[col] = prevDef
		}
	}

	return
}

func normalizeDef(s string) string {
	x := strings.TrimSpace(s)
	x = strings.Join(strings.Fields(x), " ")
	return strings.ToLower(x)
}
