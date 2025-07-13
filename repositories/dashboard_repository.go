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
	// Temporary struct to scan raw query
	type RawSeatData struct {
		ShowID      string
		Category    string
		TotalSeats  int
		BookedSeats int
	}

	var rawData []RawSeatData

	// Ganti query ini sesuai struktur tabel kamu
	err := r.DB.Table("seats").
		Select("seats.show_id, seats.category, seats.color, COUNT(seats.id) as total_seats, COUNT(booked_seats.id) as booked_seats").
		Joins("LEFT JOIN booked_seats ON seats.id = booked_seats.seat_id").
		Where("seats.category != ?", "STAGE").
		Group("seats.show_id, seats.category").
		Scan(&rawData).Error
	if err != nil {
		return nil, err
	}

	// Transform to nested map
	bookedSeats := make(map[string]map[string]models.SeatCategorySummary)
	for _, row := range rawData {
		if _, ok := bookedSeats[row.ShowID]; !ok {
			bookedSeats[row.ShowID] = make(map[string]models.SeatCategorySummary)
		}
		bookedSeats[row.ShowID][row.Category] = models.SeatCategorySummary{
			TotalSeats:  row.TotalSeats,
			BookedSeats: row.BookedSeats,
		}
	}

	return &models.DashboardSummary{
		BookedSeats: bookedSeats,
	}, nil
}
