package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Ticket struct {
	ID               string `json:"id" gorm:"primaryKey;type:uuid"`
	ExtTicketID      string `json:"ticket_id"`
	TicketCode       string  `json:"ticket_code" gorm:"uniqueIndex:idx_ticket_order"`
	OrderID          string  `json:"order_id" gorm:"uniqueIndex:idx_ticket_order"`
	Name             string  `json:"name"`
	Email            string  `json:"email"`
	Phone            string  `json:"phone"`
	Gender           string  `json:"gender"`
	Age              int     `json:"age"`
	City             string  `json:"city"`
	Province         string  `json:"province"`
	TicketName       string  `json:"ticket_name"`
	Category         string  `json:"category"`
	RegistrationTime string  `json:"registration_time"`
	BuyerName        string  `json:"buyer_name"`
	BuyerEmail       string  `json:"buyer_email"`
	BuyerPhone       string  `json:"buyer_phone"`
	VoucherName      string  `json:"voucher_name"`
	EventID          string  `json:"event_id"`
	Event            *Event      `json:"event,omitempty" gorm:"foreignKey:EventID;references:ID"`
	UserID           *string     `json:"user_id"`
	User             *User       `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
	GoodieBagClaimed bool        `json:"goodie_bag_claimed"`
	BookedSeat       *BookedSeat `json:"booked_seat,omitempty" gorm:"foreignKey:TicketID;references:ID"`
}

// Auto-generate UUID before create
func (t *Ticket) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return
}
