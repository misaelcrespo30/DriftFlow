package state

var models []interface{}

// SetModels sets the slice of models used by DriftFlow when generating
// migrations or seeds. The provided slice is copied to avoid unintended
// modifications after the call.
func SetModels(ms []interface{}) {
	models = append([]interface{}(nil), ms...)
}

// Models returns the currently configured models slice.
func Models() []interface{} {
	return append([]interface{}(nil), models...)
}
