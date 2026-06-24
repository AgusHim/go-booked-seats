package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)
type BookedSeat struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	SeatID    string    `json:"seat_id"`
	Seat      *Seat     `json:"seat,omitempty" gorm:"foreignKey:SeatID;references:ID"`
	EventID   string    `json:"event_id"`
	Event     *Event    `json:"event,omitempty" gorm:"foreignKey:EventID;references:ID"`
	AdminID   string    `json:"admin_id"`
	Name      string    `json:"name"`
	TicketID  string    `json:"ticket_id" validate:"required"`
	Ticket    *Ticket   `json:"ticket,omitempty" gorm:"foreignKey:TicketID;references:ID"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (b *BookedSeat) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return
}
