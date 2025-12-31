package seed

import (
	"github.com/Elevitae/elevitae-backend/libs/helpers"
	"github.com/misaelcrespo30/DriftFlow/internal/demo/models"

	"gorm.io/gorm"
	"log"
)

type TenantSeeder struct{}

func (s TenantSeeder) Name() string {
	return "TenantSeeder"
}

func (s TenantSeeder) Seed(db *gorm.DB, filePath string) error {
	var tenants []models.Tenant

	if err := helpers.ReadJSON(filePath, &tenants); err != nil {
		log.Printf("Error al leer JSON: %v", err)
		return err
	}

	for _, tenant := range tenants {
		tenant.TenantID = helpers.GenerateUUID()
		var existing models.Tenant
		if err := db.Where("tenant_id = ?", tenant.TenantID).FirstOrCreate(&existing, tenant).Error; err != nil {
			log.Printf("Error insertando tenant %s: %v", tenant.TenantName, err)
		}
	}

	log.Printf("%s completado.", s.Name())
	return nil
}
