// controllers/ticket_controller.go
package controllers

import (
	"go-ticketing/services"

	"github.com/gofiber/fiber/v2"
)

type DashboardController struct {
	service *services.DashboardService
}

func NewDashboardController(service *services.DashboardService) *DashboardController {
	return &DashboardController{service}
}

func (c *DashboardController) GetDashboardData(ctx *fiber.Ctx) error {
	data, err := c.service.GetDashboardData()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": data, "message": "Success get dashboard data"})
}
