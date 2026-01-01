package seed

import (
	"encoding/json"
	"os"

	models "github.com/misaelcrespo30/DriftFlow/internal/demo/models"
	"gorm.io/gorm"
)

type PlanSeeder struct{}

func (s PlanSeeder) Seed(db *gorm.DB, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var rows []models.Plan
	if err := json.Unmarshal(data, &rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	return db.Create(&rows).Error
}
