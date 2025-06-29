package state

var registeredModels []interface{}

func SetModels(models []interface{}) {
	registeredModels = append([]interface{}(nil), models...)
}

func GetModels() []interface{} {
	return append([]interface{}(nil), registeredModels...)
}
