package models

type DashboardSummary struct {
	BookedSeats map[string]map[string]SeatCategorySummary `json:"booked_seats"`
}

type SeatCategorySummary struct {
	TotalSeats  int `json:"total_seats"`
	BookedSeats int `json:"booked_seats"`
}
