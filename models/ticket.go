package models

type Ticket struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Gender     string `json:"gender"`
	TicketName string `json:"ticket_name"`
	ShowID     string `json:"show_id"`

	BookedSeat *BookedSeat `json:"booked_seat,omitempty" gorm:"foreignKey:TicketID;references:ID"`
}
