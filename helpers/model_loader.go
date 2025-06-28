package helpers

import (
	"github.com/misaelcrespo30/DriftFlow/internal/models"
	"os"
)

func LoadModels() ([]interface{}, error) {
	modelPath := os.Getenv("MODELS_PATH")
	if modelPath != "" && modelPath != "internal/models" {
		return nil, os.ErrNotExist
	}

	return models.Models(), nil
}
