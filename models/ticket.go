package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Ticket struct {
	ID         string `json:"id" gorm:"primaryKey;type:uuid"`
	TicketID   string `json:"ticket_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Gender     string `json:"gender"`
	TicketName string `json:"ticket_name"`
	ShowID     string `json:"show_id"`

	BookedSeat *BookedSeat `json:"booked_seat,omitempty" gorm:"foreignKey:TicketID;references:ID"`
}

// Auto-generate UUID before create
func (t *Ticket) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return
}
