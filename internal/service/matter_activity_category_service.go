package service

import (
	"context"

	"matters-service/internal/models"
	"matters-service/internal/repository"
)

// MatterActivityCategoryService exposes business logic for MatterActivityCategory operations.
type MatterActivityCategoryService interface {
	List(ctx context.Context) ([]models.MatterActivityCategory, error)
	Get(ctx context.Context, id uint) (models.MatterActivityCategory, error)
	Create(ctx context.Context, m models.MatterActivityCategory) (models.MatterActivityCategory, error)
}

type matterActivityCategoryService struct {
	repo repository.MatterActivityCategoryRepository
}

// NewMatterActivityCategoryService creates a new service instance.
/*func NewMatterActivityCategoryService() MatterActivityCategoryService {
	return &matterActivityCategoryService{repo: repository.NewMatterActivityCategoryRepository()}
}*/

func (s *matterActivityCategoryService) List(ctx context.Context) ([]models.MatterActivityCategory, error) {
	return s.repo.List(ctx)
}

func (s *matterActivityCategoryService) Get(ctx context.Context, id uint) (models.MatterActivityCategory, error) {
	return s.repo.Get(ctx, id)
}

func (s *matterActivityCategoryService) Create(ctx context.Context, m models.MatterActivityCategory) (models.MatterActivityCategory, error) {
	return s.repo.Create(ctx, m)
}
