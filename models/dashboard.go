package models

type DashboardSummary struct {
	BookedSeats map[string]map[string]SeatCategorySummary `json:"booked_seats"`
	TicketSummary map[string]map[string]int                 `json:"ticket_summary"`
}

type SeatCategorySummary struct {
	TotalSeats  int `json:"total_seats"`
	BookedSeats int `json:"booked_seats"`
	Color       string `json:"color"`
}
