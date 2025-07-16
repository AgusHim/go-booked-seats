package controllers

import (
	"encoding/json"
	"go-ticketing/models"
	"go-ticketing/services"

	"github.com/gofiber/fiber/v2"
)

type BookedSeatController struct {
	WS      WebsocketController
	Service *services.BookedSeatService
}

func NewBookedSeatController(service *services.BookedSeatService, ws WebsocketController) *BookedSeatController {
	return &BookedSeatController{Service: service, WS: ws}
}

func (c *BookedSeatController) GetAll(ctx *fiber.Ctx) error {
	showID := ctx.Query("show_id")
	bookedSeats, err := c.Service.GetAll(showID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": bookedSeats})
}

func (c *BookedSeatController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	bookedSeat, err := c.Service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Booked seat not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": bookedSeat})
}

func (c *BookedSeatController) Create(ctx *fiber.Ctx) error {
	var input models.BookedSeat
	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid input"})
	}
	if err := c.Service.Create(&input); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": input})
}

func (c *BookedSeatController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	bookedSeat, err := c.Service.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Booked seat not found"})
	}
	if err := ctx.BodyParser(bookedSeat); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": "Invalid input"})
	}
	if err := c.Service.Update(bookedSeat); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": bookedSeat})
}

func (c *BookedSeatController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	adminID := ctx.Locals("user_id").(string)
	if err := c.Service.Delete(id, adminID); err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Deleted"})
}

func (c *BookedSeatController) UpsertBookedSeats(ctx *fiber.Ctx) error {
	var seats []models.BookedSeat
	if err := ctx.BodyParser(&seats); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
			"error":   err.Error(),
		})
	}

	updatedSeats, err := c.Service.UpsertBookedSeats(seats)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to upsert booked seats",
			"error":   err.Error(),
		})
	}

	// Broadcast tiap seat secara paralel (goroutine per seat)
	for _, seat := range updatedSeats {
		// Copy seat ke dalam loop agar tidak race condition
		seatCopy := seat

		go func(s models.BookedSeat) {
			payload, err := json.Marshal(s)
			if err != nil {
				// Optional: log error encoding
				return
			}

			msg := models.Message{
				Type:     "booked_seat",
				SenderID: "system",
				Payload:  payload,
			}

			c.WS.SendWebsocketMessage(msg)
		}(seatCopy)
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    updatedSeats,
		"message": "Successfully upserted booked seats",
	})
}
