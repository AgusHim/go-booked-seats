package repositories

import (
	"go-ticketing/models"

	"gorm.io/gorm"
)

type DashboardRepository struct {
	DB *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{DB: db}
}

func (r *DashboardRepository) GetDashboardData() (*models.DashboardSummary, error) {
	// --- Query Seat Summary ---
	type RawSeatData struct {
		ShowID      string
		Category    string
		Color       string
		TotalSeats  int
		BookedSeats int
	}

	var rawSeatData []RawSeatData

	err := r.DB.Table("seats").
		Select("seats.show_id, seats.category, seats.color, COUNT(seats.id) as total_seats, COUNT(booked_seats.id) as booked_seats").
		Joins("LEFT JOIN booked_seats ON seats.id = booked_seats.seat_id").
		Where("seats.category != ?", "STAGE").
		Group("seats.show_id, seats.category, seats.color").
		Scan(&rawSeatData).Error
	if err != nil {
		return nil, err
	}

	bookedSeats := make(map[string]map[string]models.SeatCategorySummary)
	for _, row := range rawSeatData {
		if _, ok := bookedSeats[row.ShowID]; !ok {
			bookedSeats[row.ShowID] = make(map[string]models.SeatCategorySummary)
		}
		bookedSeats[row.ShowID][row.Category] = models.SeatCategorySummary{
			TotalSeats:  row.TotalSeats,
			BookedSeats: row.BookedSeats,
			Color:       row.Color,
		}
	}

	// --- Query Ticket Summary ---
	type RawTicketData struct {
		ShowID     string
		TicketName string
		Count      int
	}

	var rawTicketData []RawTicketData

	err = r.DB.Table("tickets").
		Select("show_id, ticket_name, COUNT(*) as count").
		Group("show_id, ticket_name").
		Scan(&rawTicketData).Error
	if err != nil {
		return nil, err
	}

	ticketSummary := make(map[string]map[string]int)
	for _, row := range rawTicketData {
		if _, ok := ticketSummary[row.ShowID]; !ok {
			ticketSummary[row.ShowID] = make(map[string]int)
		}
		ticketSummary[row.ShowID][row.TicketName] = row.Count
	}

	// --- Return all summary ---
	return &models.DashboardSummary{
		BookedSeats:   bookedSeats,
		TicketSummary: ticketSummary,
	}, nil
}
