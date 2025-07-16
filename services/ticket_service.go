// services/ticket_service.go
package services

import (
	"encoding/csv"
	"go-ticketing/models"
	"go-ticketing/repositories"
	"io"
	"mime/multipart"

	"github.com/google/uuid"
)

type TicketService interface {
	Create(ticket *models.Ticket) error
	GetAll(search string, page int, limit int, showID string) ([]models.Ticket, int64, error)
	GetByID(id string) (*models.Ticket, error)
	Update(ticket *models.Ticket) error
	Delete(id string) error
	ImportFromCSV(file multipart.File) error
}

type ticketService struct {
	repo repositories.TicketRepository
}

func NewTicketService(repo repositories.TicketRepository) TicketService {
	return &ticketService{repo}
}

func (s *ticketService) Create(ticket *models.Ticket) error {
	return s.repo.Create(ticket)
}

func (s *ticketService) GetAll(search string, page int, limit int, showID string) ([]models.Ticket, int64, error) {
	return s.repo.FindAll(search, page, limit, showID)
}

func (s *ticketService) GetByID(id string) (*models.Ticket, error) {
	return s.repo.FindByID(id)
}

func (s *ticketService) Update(ticket *models.Ticket) error {
	return s.repo.Update(ticket)
}

func (s *ticketService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *ticketService) ImportFromCSV(file multipart.File) error {
	reader := csv.NewReader(file)

	// Skip header
	_, err := reader.Read()
	if err != nil {
		return err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		ticket := models.Ticket{
			ID:         uuid.New().String(), // Optional (bisa juga andalkan BeforeCreate)
			TicketID:   record[0],
			Name:       record[1],
			Email:      record[2],
			Phone:      record[3],
			Gender:     record[4],
			TicketName: record[5],
			ShowID:     record[6],
		}

		if err := s.repo.Create(&ticket); err != nil {
			return err
		}
	}

	return nil
}
