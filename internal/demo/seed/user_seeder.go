package seed

import (
	"github.com/misaelcrespo30/DriftFlow/helpers"
	"github.com/misaelcrespo30/DriftFlow/internal/demo/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
)

type UserSeeder struct{}

func (s UserSeeder) Name() string {
	return "UserSeeder"
}

func (s UserSeeder) Seed(db *gorm.DB, filePath string) error {
	var usersJSON []models.User

	// Leer archivo JSON
	if err := helpers.ReadJSON(filePath, &usersJSON); err != nil {
		log.Printf("Error al leer JSON: %v", err)
		return err
	}

	for _, userJSON := range usersJSON {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userJSON.PasswordHash), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error al hashear contraseña: %v", err)
			continue
		}
		stamp := uuid.NewString()
		user := models.User{
			//UserId:            helpers.GenerateUUID(),
			UserID:       helpers.GenerateUUID(),
			Email:        userJSON.Email,
			UserName:     userJSON.UserName,
			PasswordHash: string(hashedPassword),
			//Role:              userJSON.Role,
			Phone:             userJSON.Phone,
			IsPhoneConfirmed:  userJSON.IsPhoneConfirmed,
			IsEmailConfirmed:  userJSON.IsEmailConfirmed,
			AccessFailedCount: userJSON.AccessFailedCount,
			IsLockoutEnabled:  userJSON.IsLockoutEnabled,
			LockoutEnd:        userJSON.LockoutEnd,
			//IsOrganization:    userJSON.IsOrganization,
			//SecurityStamp: uuid.NewString(),
			SecurityStamp: &stamp,
		}

		// Insertar con FirstOrCreate
		var existing models.User
		if err := db.Where("email = ?", user.Email).FirstOrCreate(&existing, user).Error; err != nil {
			log.Printf("Error insertando usuario %s: %v", user.Email, err)
		}
	}

	log.Printf("%s completado.", s.Name())
	return nil
}
