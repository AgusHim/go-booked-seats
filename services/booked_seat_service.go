package services

import (
	"errors"
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

func (s *BookedSeatService) Delete(id string, sessionAdminID string) error {
	// Ambil data booked seat dulu
	bookedSeat, err := s.Repo.FindByID(id)
	if err != nil {
		return err
	}

	// Cek apakah admin_id sesuai dengan session
	if bookedSeat.AdminID != sessionAdminID {
		return errors.New("not authorize to delete")
	}

	return s.Repo.Delete(id)
}

func (s *BookedSeatService) UpsertBookedSeats(seats []models.BookedSeat) ([]models.BookedSeat, error) {
	return s.Repo.UpsertBookedSeats(seats)
}
