package seed

import (
	"github.com/misaelcrespo30/DriftFlow/helpers"
	"github.com/misaelcrespo30/DriftFlow/internal/demo/models"
	"gorm.io/gorm"
	"log"
)

type PlanSeeder struct{}

func (s PlanSeeder) Name() string {
	return "PlanSeeder"
}

func (s PlanSeeder) Seed(db *gorm.DB, filePath string) error {
	var plansJson []models.Plan

	// Leer archivo JSON
	if err := helpers.ReadJSON(filePath, &plansJson); err != nil {
		log.Printf("Error al leer plans.json: %v", err)
		return err
	}

	for _, planJson := range plansJson {

		plan := models.Plan{
			Name:       planJson.Name,
			ExternalID: planJson.ExternalID,
			MaxSeats:   planJson.MaxSeats,
			MinSeats:   planJson.MinSeats,
			IsDisabled: planJson.IsDisabled,
			Version:    planJson.Version,
		}

		var existing models.Plan
		// Insertar si no existe, basado en ID
		if err := db.Where("Name = ?", plan.Name).FirstOrCreate(&existing, plan).Error; err != nil {
			log.Printf("Error insertando plan %s (Name %d): %v", plan.Name, err)
		}
	}

	log.Printf("%s completado.", s.Name())
	return nil
}
