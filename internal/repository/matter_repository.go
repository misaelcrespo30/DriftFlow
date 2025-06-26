package repository

import (
	"context"

	"gorm.io/gorm"

	"matters-service/internal/models"
)

// MatterRepository defines database operations for Matter using GORM.

type MatterRepository interface {
	List(ctx context.Context) ([]models.Matter, error)
	Get(ctx context.Context, id uint) (models.Matter, error)
	Create(ctx context.Context, m models.Matter) (models.Matter, error)
}

type gormMatterRepo struct {
	db *gorm.DB
}

// NewMatterRepository creates a repository using a new GORM connection based on configuration.
/*func NewMatterRepository() MatterRepository {
	cfg := config.LoadConfig()
	db, err := database.OpenGorm(*cfg)
	if err != nil {
		panic(err)
	}
	return &gormMatterRepo{db: db}
}*/

// NewMatterRepositoryWithDB allows injecting an existing GORM instance.
func NewMatterRepositoryWithDB(db *gorm.DB) MatterRepository {
	return &gormMatterRepo{db: db}
}

func (r *gormMatterRepo) List(ctx context.Context) ([]models.Matter, error) {
	var matters []models.Matter
	if err := r.db.WithContext(ctx).Find(&matters).Error; err != nil {
		return nil, err
	}
	return matters, nil
}

func (r *gormMatterRepo) Get(ctx context.Context, id uint) (models.Matter, error) {
	var m models.Matter
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return models.Matter{}, err
	}
	return m, nil
}

func (r *gormMatterRepo) Create(ctx context.Context, m models.Matter) (models.Matter, error) {
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return models.Matter{}, err
	}
	return m, nil
}
