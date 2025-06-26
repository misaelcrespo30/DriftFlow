package service

import (
	"context"

	"matters-service/internal/models"
	"matters-service/internal/repository"
)

type MatterService interface {
	List(ctx context.Context) ([]models.Matter, error)
	Get(ctx context.Context, id uint) (models.Matter, error)
	Create(ctx context.Context, m models.Matter) (models.Matter, error)
}

type matterService struct {
	repo repository.MatterRepository
}

/*func NewMatterService() MatterService {
	return &matterService{repo: repository.NewMatterRepository()}
}*/

func (s *matterService) List(ctx context.Context) ([]models.Matter, error) {
	return s.repo.List(ctx)
}

func (s *matterService) Get(ctx context.Context, id uint) (models.Matter, error) {
	return s.repo.Get(ctx, id)
}

func (s *matterService) Create(ctx context.Context, m models.Matter) (models.Matter, error) {
	return s.repo.Create(ctx, m)
}
