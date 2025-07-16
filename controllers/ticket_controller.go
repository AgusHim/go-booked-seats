// controllers/ticket_controller.go
package controllers

import (
	"go-ticketing/models"
	"go-ticketing/services"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type TicketController struct {
	service services.TicketService
}

func NewTicketController(service services.TicketService) *TicketController {
	return &TicketController{service}
}

func (c *TicketController) Create(ctx *fiber.Ctx) error {
	var ticket models.Ticket
	if err := ctx.BodyParser(&ticket); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if err := c.service.Create(&ticket); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": ticket, "message": "Ticket created"})
}

func (c *TicketController) GetAll(ctx *fiber.Ctx) error {
	search := ctx.Query("search", "")
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	showID := ctx.Query("show_id", "")
	tickets, total, err := c.service.GetAll(search, page, limit, showID)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	totalPages := (total + int64(limit) - 1) / int64(limit)

	return ctx.JSON(fiber.Map{"success": true, "data": tickets, "message": "Success get data", "meta": fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}})
}

func (c *TicketController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	ticket, err := c.service.GetByID(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Ticket not found"})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": ticket, "message": "Success get data"})
}

func (c *TicketController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var ticket models.Ticket
	if err := ctx.BodyParser(&ticket); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	ticket.ID = id
	if err := c.service.Update(&ticket); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": ticket, "message": "Ticket updated"})
}

func (c *TicketController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.Delete(id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Ticket deleted"})
}
