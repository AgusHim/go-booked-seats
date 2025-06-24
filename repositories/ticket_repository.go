// repositories/ticket_repository.go
package repositories

import (
	"go-ticketing/models"
	"strings"

	"gorm.io/gorm"
)

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	FindAll(search string, page int, limit int) ([]models.Ticket, int64, error)
	FindByID(id string) (*models.Ticket, error)
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

func (r *ticketRepository) FindAll(search string, page int, limit int) ([]models.Ticket, int64, error) {
	var tickets []models.Ticket
	var total int64

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := r.db.Model(&models.Ticket{})

	if search != "" {
		lowerKeyword := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			r.db.Where("LOWER(id) LIKE ?", lowerKeyword).
				Or("LOWER(name) LIKE ?", lowerKeyword).
				Or("LOWER(email) LIKE ?", lowerKeyword),
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

func (r *ticketRepository) Update(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}

func (r *ticketRepository) Delete(id string) error {
	return r.db.Delete(&models.Ticket{}, "id = ?", id).Error
}
