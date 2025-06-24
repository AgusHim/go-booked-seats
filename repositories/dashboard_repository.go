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
	var result []models.DashboardData
	err := r.DB.Table("booked_seats").
		Select("show_id, COUNT(*) as total_booked_seat").
		Group("show_id").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	var totalTicket int64
	if err := r.DB.Model(&models.Ticket{}).Count(&totalTicket).Error; err != nil {
		return nil, err
	}

	var totalTicketBooked int64
	err = r.DB.
		Table("booked_seats").
		Select("COUNT(DISTINCT ticket_id)").
		Scan(&totalTicketBooked).Error
	if err != nil {
		return nil, err
	}

	summary := &models.DashboardSummary{
		BookedSeatPerShow:   result,
		TotalTicket:         totalTicket,
		TotalTicketBooked:   totalTicketBooked,
		TotalTicketUnbooked: totalTicket - totalTicketBooked,
	}

	return summary, nil
}
