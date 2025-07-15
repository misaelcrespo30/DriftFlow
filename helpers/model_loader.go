package helpers

import (
	"errors"
	"github.com/misaelcrespo30/DriftFlow/state"
)

// LoadModels validates the directory defined by the MODELS_PATH environment
// variable and returns the compiled model instances. It ensures at least one
// exported struct exists in that directory.
func LoadModels() ([]interface{}, error) {

	models := state.GetModels()
	if len(models) == 0 {
		return nil, errors.New("no models registered: call state.SetModels(...) before executing DriftFlow commands")
	}
	return models, nil

}
