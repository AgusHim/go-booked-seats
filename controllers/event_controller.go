package controllers

import (
	"go-ticketing/models"
	"go-ticketing/services"

	"github.com/gofiber/fiber/v2"
)

type EventController struct {
	service services.EventService
}

func NewEventController(service services.EventService) *EventController {
	return &EventController{service}
}

func (c *EventController) GetEvents(ctx *fiber.Ctx) error {
	events, err := c.service.GetEvents()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": events, "message": "Success get events"})
}

func (c *EventController) GetEvent(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	event, err := c.service.GetEvent(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Event not found"})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": event, "message": "Success get event"})
}

func (c *EventController) UpdateEvent(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var event models.Event
	if err := ctx.BodyParser(&event); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	event.ID = id
	if err := c.service.UpdateEvent(&event); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": event, "message": "Event updated"})
}

func (c *EventController) CreateEvent(ctx *fiber.Ctx) error {
	var event models.Event
	if err := ctx.BodyParser(&event); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if err := c.service.CreateEvent(&event); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": event, "message": "Event created"})
}

func (c *EventController) DeleteEvent(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.DeleteEvent(id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Event deleted"})
}
