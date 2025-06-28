package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/misaelcrespo30/DriftFlow/internal/models"
)

// MatterActivityCategoryRepository defines DB operations for MatterActivityCategory using GORM.
type MatterActivityCategoryRepository interface {
	List(ctx context.Context) ([]models.MatterActivityCategory, error)
	Get(ctx context.Context, id uint) (models.MatterActivityCategory, error)
	Create(ctx context.Context, m models.MatterActivityCategory) (models.MatterActivityCategory, error)
}

type gormCategoryRepo struct {
	db *gorm.DB
}

/*func NewMatterActivityCategoryRepository() MatterActivityCategoryRepository {
	cfg := config.LoadConfig()
	db, err := database.OpenGorm(*cfg)
	if err != nil {
		panic(err)
	}
	return &gormCategoryRepo{db: db}
}*/

func NewMatterActivityCategoryRepositoryWithDB(db *gorm.DB) MatterActivityCategoryRepository {
	return &gormCategoryRepo{db: db}
}

func (r *gormCategoryRepo) List(ctx context.Context) ([]models.MatterActivityCategory, error) {
	var categories []models.MatterActivityCategory
	if err := r.db.WithContext(ctx).Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *gormCategoryRepo) Get(ctx context.Context, id uint) (models.MatterActivityCategory, error) {
	var c models.MatterActivityCategory
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return models.MatterActivityCategory{}, err
	}
	return c, nil
}

func (r *gormCategoryRepo) Create(ctx context.Context, m models.MatterActivityCategory) (models.MatterActivityCategory, error) {
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return models.MatterActivityCategory{}, err
	}
	return m, nil
}
