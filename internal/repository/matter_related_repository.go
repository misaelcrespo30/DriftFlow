package repository

import (
	"context"

	"gorm.io/gorm"

	"matters-service/internal/models"
)

// MatterRelatedRepository defines database operations for MatterRelated using GORM.
type MatterRelatedRepository interface {
	List(ctx context.Context) ([]models.MatterRelated, error)
	Get(ctx context.Context, id uint) (models.MatterRelated, error)
	Create(ctx context.Context, m models.MatterRelated) (models.MatterRelated, error)
}

type gormRelatedRepo struct {
	db *gorm.DB
}

/*func NewMatterRelatedRepository() MatterRelatedRepository {
	cfg := config.LoadConfig()
	db, err := database.OpenGorm(*cfg)
	if err != nil {
		panic(err)
	}
	return &gormRelatedRepo{db: db}
}*/

func NewMatterRelatedRepositoryWithDB(db *gorm.DB) MatterRelatedRepository {
	return &gormRelatedRepo{db: db}
}

func (r *gormRelatedRepo) List(ctx context.Context) ([]models.MatterRelated, error) {
	var rel []models.MatterRelated
	if err := r.db.WithContext(ctx).Find(&rel).Error; err != nil {
		return nil, err
	}
	return rel, nil
}

func (r *gormRelatedRepo) Get(ctx context.Context, id uint) (models.MatterRelated, error) {
	var m models.MatterRelated
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		return models.MatterRelated{}, err
	}
	return m, nil
}

func (r *gormRelatedRepo) Create(ctx context.Context, m models.MatterRelated) (models.MatterRelated, error) {
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return models.MatterRelated{}, err
	}
	return m, nil
}
