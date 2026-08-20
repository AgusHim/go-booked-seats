// repositories/ticket_repository.go
package repositories

import (
	"go-ticketing/models"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	FindAll(search string, page int, limit int, show_id string) ([]models.Ticket, int64, error)
	FindByID(id string) (*models.Ticket, error)
	FindByTicketCode(ticketCode string) (*models.Ticket, error)
	ToggleGoodieBag(id string) (*models.Ticket, error)
	MarkGoodieBagsClaimed(ids []string) ([]models.Ticket, []models.Ticket, error)
	UpdateDarisiniScanLog(id string, status string, response string, scannedAt time.Time) error
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

func (r *ticketRepository) FindAll(search string, page int, limit int, eventID string) ([]models.Ticket, int64, error) {
	var tickets []models.Ticket
	var total int64

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := r.db.Model(&models.Ticket{}).Preload("BookedSeat").Preload("BookedSeat.Seat").Preload("Event")

	if eventID != "" {
		query = query.Where("event_id = ?", eventID)
	}

	if search != "" {
		lowerKeyword := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(ticket_code) LIKE ? OR LOWER(ext_ticket_id) LIKE ? OR LOWER(name) LIKE ? OR LOWER(email) LIKE ?",
			lowerKeyword, lowerKeyword, lowerKeyword, lowerKeyword,
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
	lookup := normalizeTicketLookup(ticketCode)
	err := r.db.Where(
		"UPPER(TRIM(ticket_code)) = ? OR UPPER(TRIM(ext_ticket_id)) = ?",
		lookup,
		lookup,
	).First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func normalizeTicketLookup(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "[]")
	return strings.ToUpper(strings.TrimSpace(value))
}

func (r *ticketRepository) ToggleGoodieBag(id string) (*models.Ticket, error) {
	var ticket models.Ticket
	if err := r.db.Preload("Event").First(&ticket, "id = ?", id).Error; err != nil {
		return nil, err
	}
	ticket.GoodieBagClaimed = !ticket.GoodieBagClaimed
	if err := r.db.Save(&ticket).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) MarkGoodieBagsClaimed(ids []string) ([]models.Ticket, []models.Ticket, error) {
	if len(ids) == 0 {
		return []models.Ticket{}, []models.Ticket{}, nil
	}

	var tickets []models.Ticket
	var newlyClaimed []models.Ticket

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&newlyClaimed).
			Clauses(clause.Returning{}).
			Where("id IN ? AND goodie_bag_claimed = ?", ids, false).
			Updates(map[string]interface{}{
				"goodie_bag_claimed":     true,
				"darisini_scan_status":   "pending",
				"darisini_scan_response": "Scan Darisini queued",
			}).Error; err != nil {
			return err
		}

		if err := tx.Preload("Event").Where("id IN ?", ids).Find(&tickets).Error; err != nil {
			return err
		}
		if len(newlyClaimed) == 0 {
			return nil
		}

		newlyClaimedIDs := make([]string, 0, len(newlyClaimed))
		for _, ticket := range newlyClaimed {
			newlyClaimedIDs = append(newlyClaimedIDs, ticket.ID)
		}
		return tx.Preload("Event").Where("id IN ?", newlyClaimedIDs).Find(&newlyClaimed).Error
	})
	if err != nil {
		return nil, nil, err
	}

	return tickets, newlyClaimed, nil
}

func (r *ticketRepository) UpdateDarisiniScanLog(id string, status string, response string, scannedAt time.Time) error {
	return r.db.Model(&models.Ticket{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"darisini_scan_status":   status,
			"darisini_scan_response": response,
			"darisini_scanned_at":    scannedAt,
		}).Error
}

func (r *ticketRepository) Update(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}

func (r *ticketRepository) Delete(id string) error {
	return r.db.Delete(&models.Ticket{}, "id = ?", id).Error
}
