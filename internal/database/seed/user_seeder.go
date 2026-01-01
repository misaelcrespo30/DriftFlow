package seed

import (
	"encoding/json"
	"fmt"
	"os"

	models "github.com/misaelcrespo30/DriftFlow/internal/demo/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserSeeder struct{}

type userSeed struct {
	models.User
	Password string `json:"password_hash"`
}

func (s UserSeeder) Seed(db *gorm.DB, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var seeds []userSeed
	if err := json.Unmarshal(data, &seeds); err != nil {
		return err
	}
	if len(seeds) == 0 {
		return nil
	}

	items := make([]models.User, 0, len(seeds))
	for _, seed := range seeds {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(seed.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hashing password for user %s: %w", seed.Email, err)
		}
		seed.User.PasswordHash = string(hashedPassword)
		items = append(items, seed.User)
	}

	return db.Create(&items).Error
}
