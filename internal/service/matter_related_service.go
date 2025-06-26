package service

import (
	"context"

	"matters-service/internal/models"
	"matters-service/internal/repository"
)

// MatterRelatedService exposes business logic for MatterRelated operations.
type MatterRelatedService interface {
	List(ctx context.Context) ([]models.MatterRelated, error)
	Get(ctx context.Context, id uint) (models.MatterRelated, error)
	Create(ctx context.Context, m models.MatterRelated) (models.MatterRelated, error)
}

type matterRelatedService struct {
	repo repository.MatterRelatedRepository
}

/*func NewMatterRelatedService() MatterRelatedService {
	return &matterRelatedService{repo: repository.NewMatterRelatedRepository()}
}*/

func (s *matterRelatedService) List(ctx context.Context) ([]models.MatterRelated, error) {
	return s.repo.List(ctx)
}

func (s *matterRelatedService) Get(ctx context.Context, id uint) (models.MatterRelated, error) {
	return s.repo.Get(ctx, id)
}

func (s *matterRelatedService) Create(ctx context.Context, m models.MatterRelated) (models.MatterRelated, error) {
	return s.repo.Create(ctx, m)
}
