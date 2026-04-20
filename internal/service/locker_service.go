package service

import (
	"context"
	"errors"
	"locker/internal/domain"
	"locker/internal/repository"
)

type LockerService struct {
	repo *repository.LockerRepository
}

func NewLockerService(repo *repository.LockerRepository) *LockerService {
	return &LockerService{repo: repo}
}

func (s *LockerService) GetLocker(ctx context.Context, id int64) (*domain.Locker, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *LockerService) GetLocationLockers(ctx context.Context, locationID int64) ([]*domain.Locker, error) {
	return s.repo.GetByLocationID(ctx, locationID)
}

func (s *LockerService) GetFreeLocker(ctx context.Context, locationID int64, size domain.LockerSize) (*domain.Locker, error) {
	locker, err := s.repo.GetFreeBySize(ctx, locationID, size)
	if err != nil {
		return nil, errors.New("no free locker available")
	}
	return locker, nil
}

func (s *LockerService) CreateLocker(ctx context.Context, locker *domain.Locker) (int64, error) {
	locker.Status = domain.LockerStatusFree
	return s.repo.Create(ctx, locker)
}
