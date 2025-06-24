// services/ticket_service.go
package services

import (
	"go-ticketing/models"
	"go-ticketing/repositories"
)

type DashboardService struct {
	repo *repositories.DashboardRepository
}

func NewDashboardService(repo *repositories.DashboardRepository) *DashboardService {
	return &DashboardService{repo}
}

func (s *DashboardService) GetDashboardData() (*models.DashboardSummary, error) {
	return s.repo.GetDashboardData()
}
