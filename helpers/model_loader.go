package helpers

import (
	"fmt"
	"path/filepath"
	"plugin"
)

// LoadModels loads a plugin named models.so in dir and returns the Models symbol.
// The plugin must export a variable or function named "Models" of type
// []interface{} or *[]interface{}.
func LoadModels(dir string) ([]interface{}, error) {
	p, err := plugin.Open(filepath.Join(dir, "models.so"))
	if err != nil {
		return nil, err
	}
	sym, err := p.Lookup("Models")
	if err != nil {
		return nil, err
	}
	switch m := sym.(type) {
	case []interface{}:
		return m, nil
	case *[]interface{}:
		return *m, nil
	default:
		return nil, fmt.Errorf("invalid Models symbol")
	}
}
