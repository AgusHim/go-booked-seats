// repositories/ticket_repository.go
package repositories

import (
	"go-ticketing/models"
	"strings"

	"gorm.io/gorm"
)

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	FindAll(search string, page int, limit int, show_id string) ([]models.Ticket, int64, error)
	FindByID(id string) (*models.Ticket, error)
	FindByTicketCode(ticketCode string) (*models.Ticket, error)
	ToggleGoodieBag(id string) (*models.Ticket, error)
	Update(ticket *models.Ticket) error
	Delete(id string) error
}

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db}
}

func (r *ticketRepository) Create(ticket *models.Ticket) error {
	return r.db.Create(ticket).Error
}

func (r *ticketRepository) FindAll(search string, page int, limit int, showID string) ([]models.Ticket, int64, error) {
	var tickets []models.Ticket
	var total int64

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := r.db.Model(&models.Ticket{}).Preload("BookedSeat").Preload("BookedSeat.Seat")

	if showID != "" {
		query = query.Where("event_id = ?", showID)
	}

	if search != "" {
		lowerKeyword := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(ticket_id) LIKE ? OR LOWER(name) LIKE ? OR LOWER(email) LIKE ?",
			lowerKeyword, lowerKeyword, lowerKeyword,
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Find(&tickets).Error
	return tickets, total, err
}

func (r *ticketRepository) FindByID(id string) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.First(&ticket, "id = ?", id).Error
	return &ticket, err
}

func (r *ticketRepository) FindByTicketCode(ticketCode string) (*models.Ticket, error) {
	var ticket models.Ticket
	err := r.db.Where("ticket_code = ?", ticketCode).First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) ToggleGoodieBag(id string) (*models.Ticket, error) {
	var ticket models.Ticket
	if err := r.db.First(&ticket, "id = ?", id).Error; err != nil {
		return nil, err
	}
	ticket.GoodieBagClaimed = !ticket.GoodieBagClaimed
	if err := r.db.Save(&ticket).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) Update(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}

func (r *ticketRepository) Delete(id string) error {
	return r.db.Delete(&models.Ticket{}, "id = ?", id).Error
}
