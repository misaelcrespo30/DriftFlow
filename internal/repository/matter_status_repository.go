package repository

import (
	"context"

	"gorm.io/gorm"

	"matters-service/internal/models"
)

// MatterStatusRepository defines database operations for MatterStatus using GORM.
type MatterStatusRepository interface {
	List(ctx context.Context) ([]models.MatterStatus, error)
	Get(ctx context.Context, id uint) (models.MatterStatus, error)
	Create(ctx context.Context, m models.MatterStatus) (models.MatterStatus, error)
}

type gormStatusRepo struct {
	db *gorm.DB
}

/*func NewMatterStatusRepository() MatterStatusRepository {
	cfg := config.LoadConfig()
	db, err := database.OpenGorm(*cfg)
	if err != nil {
		panic(err)
	}
	return &gormStatusRepo{db: db}
}*/

func NewMatterStatusRepositoryWithDB(db *gorm.DB) MatterStatusRepository {
	return &gormStatusRepo{db: db}
}

func (r *gormStatusRepo) List(ctx context.Context) ([]models.MatterStatus, error) {
	var statuses []models.MatterStatus
	if err := r.db.WithContext(ctx).Find(&statuses).Error; err != nil {
		return nil, err
	}
	return statuses, nil
}

func (r *gormStatusRepo) Get(ctx context.Context, id uint) (models.MatterStatus, error) {
	var s models.MatterStatus
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return models.MatterStatus{}, err
	}
	return s, nil
}

func (r *gormStatusRepo) Create(ctx context.Context, m models.MatterStatus) (models.MatterStatus, error) {
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return models.MatterStatus{}, err
	}
	return m, nil
}
