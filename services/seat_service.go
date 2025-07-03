package services

import (
	"context"
	"go-ticketing/models"
	"go-ticketing/repositories"
)

type SeatService interface {
	GetAll(showID string) ([]models.Seat, error)
	GetByID(id string) (models.Seat, error)
	Create(seat models.Seat) error
	Update(seat models.Seat) error
	Delete(id string) error
	TryLockSeat(ctx context.Context, showID string, seatID string, userID string) (string, error)
	GetLockedSeats(ctx context.Context, showID string) ([]*models.BookedSeat, error)
}

type seatService struct {
	repo repositories.SeatRepository
}

func NewSeatService(repo repositories.SeatRepository) SeatService {
	return &seatService{repo}
}

func (s *seatService) GetAll(showID string) ([]models.Seat, error) {
	return s.repo.FindAll(showID)
}

func (s *seatService) GetByID(id string) (models.Seat, error) {
	return s.repo.FindByID(id)
}

func (s *seatService) Create(seat models.Seat) error {
	return s.repo.Create(seat)
}

func (s *seatService) Update(seat models.Seat) error {
	return s.repo.Update(seat)
}

func (s *seatService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *seatService) TryLockSeat(ctx context.Context, showID string, seatID string, userID string) (string, error) {
	return s.repo.LockSeat(ctx, showID, seatID, userID)
}

func (s *seatService) GetLockedSeats(ctx context.Context, showID string) ([]*models.BookedSeat, error) {
	return s.repo.GetLockedSeats(ctx, showID)
}
