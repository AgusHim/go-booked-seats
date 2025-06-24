package models

type DashboardData struct {
	ShowID          string `json:"show_id"`
	TotalBookedSeat int64  `json:"total_booked_seat"`
}

type DashboardSummary struct {
	BookedSeatPerShow   []DashboardData `json:"booked_seat_per_show"`
	TotalTicket         int64           `json:"total_ticket"`
	TotalTicketBooked   int64           `json:"total_ticket_booked"`
	TotalTicketUnbooked int64           `json:"total_ticket_unbooked"`
}
