package repository

import (
	"context"

	"gorm.io/gorm"

	"matters-service/internal/models"
)

// MatterActivityRepository defines database operations for MatterActivity using GORM.
type MatterActivityRepository interface {
	List(ctx context.Context) ([]models.MatterActivity, error)
	Get(ctx context.Context, id uint) (models.MatterActivity, error)
	Create(ctx context.Context, m models.MatterActivity) (models.MatterActivity, error)
}

type gormActivityRepo struct {
	db *gorm.DB
}

/*func NewMatterActivityRepository() MatterActivityRepository {
	cfg := config.LoadConfig()
	db, err := database.OpenGorm(*cfg)
	if err != nil {
		panic(err)
	}
	return &gormActivityRepo{db: db}
}*/

func NewMatterActivityRepositoryWithDB(db *gorm.DB) MatterActivityRepository {
	return &gormActivityRepo{db: db}
}

func (r *gormActivityRepo) List(ctx context.Context) ([]models.MatterActivity, error) {
	var acts []models.MatterActivity
	if err := r.db.WithContext(ctx).Find(&acts).Error; err != nil {
		return nil, err
	}
	return acts, nil
}

func (r *gormActivityRepo) Get(ctx context.Context, id uint) (models.MatterActivity, error) {
	var a models.MatterActivity
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return models.MatterActivity{}, err
	}
	return a, nil
}

func (r *gormActivityRepo) Create(ctx context.Context, a models.MatterActivity) (models.MatterActivity, error) {
	if err := r.db.WithContext(ctx).Create(&a).Error; err != nil {
		return models.MatterActivity{}, err
	}
	return a, nil
}
