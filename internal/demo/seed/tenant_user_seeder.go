package seed

import (
	"github.com/misaelcrespo30/DriftFlow/helpers"
	"github.com/misaelcrespo30/DriftFlow/internal/demo/models"

	"gorm.io/gorm"
	"log"
)

type TenantUserSeeder struct{}

func (s TenantUserSeeder) Name() string {
	return "TenantUserSeeder"
}

func (s TenantUserSeeder) Seed(db *gorm.DB, filePath string) error {
	var itemsJSON []models.TenantUser

	if err := helpers.ReadJSON(filePath, &itemsJSON); err != nil {
		log.Printf("Error al leer JSON: %v", err)
		return err
	}

	for _, itemJSON := range itemsJSON {

		ObjUser := helpers.GetRandomRecord(db, &models.User{})
		ObjTenant := helpers.GetRandomRecord(db, &models.Tenant{})

		tenantUser := models.TenantUser{
			UserID:             ObjUser.UserID,
			TenantID:           ObjTenant.TenantID,
			Relationship:       itemJSON.Relationship,
			IsActive:           itemJSON.IsActive,
			IsDefault:          itemJSON.IsDefault,
			OriginatedUser:     itemJSON.OriginatedUser,
			ExternalIdentityID: itemJSON.ExternalIdentityID,
		}

		var existing models.TenantUser
		//	if err := db.Where("user_id = ? AND tenant_id = ?", tenantUser.UserId, tenantUser.TenantID).
		if err := db.Where("user_id = ? AND tenant_id = ?", tenantUser.UserID, tenantUser.TenantID).
			FirstOrCreate(&existing, tenantUser).Error; err != nil {
			log.Printf("Error insertando TenantUser para %s/%s: %v", ObjUser.Email, ObjTenant.TenantName, err)
		}
	}

	log.Printf("%s completado.", s.Name())
	return nil
}
