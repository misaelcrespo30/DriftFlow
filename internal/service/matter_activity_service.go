package service

import (
	"context"

	"matters-service/internal/models"
	"matters-service/internal/repository"
)

// MatterActivityService exposes business logic for MatterActivity operations.
type MatterActivityService interface {
	List(ctx context.Context) ([]models.MatterActivity, error)
	Get(ctx context.Context, id uint) (models.MatterActivity, error)
	Create(ctx context.Context, m models.MatterActivity) (models.MatterActivity, error)
}

type matterActivityService struct {
	repo repository.MatterActivityRepository
}

/*func NewMatterActivityService() MatterActivityService {
	return &matterActivityService{repo: repository.NewMatterActivityRepository()}
}*/

func (s *matterActivityService) List(ctx context.Context) ([]models.MatterActivity, error) {
	return s.repo.List(ctx)
}

func (s *matterActivityService) Get(ctx context.Context, id uint) (models.MatterActivity, error) {
	return s.repo.Get(ctx, id)
}

func (s *matterActivityService) Create(ctx context.Context, m models.MatterActivity) (models.MatterActivity, error) {
	return s.repo.Create(ctx, m)
}
