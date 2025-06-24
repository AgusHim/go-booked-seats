package services

import (
	"go-ticketing/models"
	"go-ticketing/repositories"
)

type BookedSeatService struct {
	Repo *repositories.BookedSeatRepository
}

func NewBookedSeatService(repo *repositories.BookedSeatRepository) *BookedSeatService {
	return &BookedSeatService{Repo: repo}
}

func (s *BookedSeatService) GetAll(showID string) ([]models.BookedSeat, error) {
	return s.Repo.FindAll(showID)
}

func (s *BookedSeatService) GetByID(id string) (*models.BookedSeat, error) {
	return s.Repo.FindByID(id)
}

func (s *BookedSeatService) Create(bookedSeat *models.BookedSeat) error {
	return s.Repo.Create(bookedSeat)
}

func (s *BookedSeatService) Update(bookedSeat *models.BookedSeat) error {
	return s.Repo.Update(bookedSeat)
}

func (s *BookedSeatService) Delete(id string) error {
	return s.Repo.Delete(id)
}

func (s *BookedSeatService) UpsertBookedSeats(seats []models.BookedSeat) ([]models.BookedSeat, error) {
	return s.Repo.UpsertBookedSeats(seats)
}
