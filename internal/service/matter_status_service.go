package service

import (
	"context"

	"github.com/misaelcrespo30/DriftFlow/internal/models"
	"github.com/misaelcrespo30/DriftFlow/internal/repository"
)

// MatterStatusService exposes business logic for MatterStatus operations.
type MatterStatusService interface {
	List(ctx context.Context) ([]models.MatterStatus, error)
	Get(ctx context.Context, id uint) (models.MatterStatus, error)
	Create(ctx context.Context, m models.MatterStatus) (models.MatterStatus, error)
}

type matterStatusService struct {
	repo repository.MatterStatusRepository
}

// NewMatterStatusService creates a new MatterStatusService.
/*func NewMatterStatusService() MatterStatusService {
	return &matterStatusService{repo: repository.NewMatterStatusRepository()}
}*/

func (s *matterStatusService) List(ctx context.Context) ([]models.MatterStatus, error) {
	return s.repo.List(ctx)
}

func (s *matterStatusService) Get(ctx context.Context, id uint) (models.MatterStatus, error) {
	return s.repo.Get(ctx, id)
}

func (s *matterStatusService) Create(ctx context.Context, m models.MatterStatus) (models.MatterStatus, error) {
	return s.repo.Create(ctx, m)
}
