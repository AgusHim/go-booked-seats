// services/ticket_service.go
package services

import (
	"go-ticketing/models"
	"go-ticketing/repositories"
)

type TicketService interface {
	Create(ticket *models.Ticket) error
	GetAll(search string, page int, limit int, showID string) ([]models.Ticket, int64, error)
	GetByID(id string) (*models.Ticket, error)
	Update(ticket *models.Ticket) error
	Delete(id string) error
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
