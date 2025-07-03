package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ticketing/models"
	"go-ticketing/services"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type SeatController struct {
	ws          WebsocketController
	seatService services.SeatService
}

func NewSeatController(s services.SeatService, ws WebsocketController) *SeatController {
	return &SeatController{seatService: s, ws: ws}
}

func (c *SeatController) GetAll(ctx *fiber.Ctx) error {
	showID := ctx.Query("show_id")
	seats, err := c.seatService.GetAll(showID)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": seats})
}

func (c *SeatController) GetByID(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	seat, err := c.seatService.GetByID(id)
	if err != nil {
		return ctx.Status(404).JSON(fiber.Map{"success": false, "message": "Seat not found"})
	}
	return ctx.JSON(fiber.Map{"success": true, "data": seat})
}

func (c *SeatController) Create(ctx *fiber.Ctx) error {
	var seat models.Seat
	if err := ctx.BodyParser(&seat); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	err := c.seatService.Create(seat)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Seat created"})
}

func (c *SeatController) Update(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var seat models.Seat
	if err := ctx.BodyParser(&seat); err != nil {
		return ctx.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	seat.ID = id
	err := c.seatService.Update(seat)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Seat updated"})
}

func (c *SeatController) Delete(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	err := c.seatService.Delete(id)
	if err != nil {
		return ctx.Status(500).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{"success": true, "message": "Seat deleted"})
}

func (ctl *SeatController) LockSeat(c *fiber.Ctx) error {
	type Request struct {
		ShowID  string `json:"show_id"`
		SeatID  string `json:"seat_id"`
		AdminID string `json:"admin_id"`
	}
	var body Request
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request",
		})
	}

	status, err := ctl.seatService.TryLockSeat(context.Background(), body.ShowID, body.SeatID, body.AdminID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Failed to lock seat",
			"error":   err.Error(),
		})
	}

	if status == "locked" || status == "unlocked" {
		payload, _ := json.Marshal(body)

		msg := models.Message{
			Type:     fmt.Sprintf("seat_%s", status),
			SenderID: "system",
			Payload:  payload,
		}

		go ctl.ws.SendWebsocketMessage(msg)
	}

	res_msg := map[string]string{
		"locked":   "Seat locked",
		"unlocked": "Seat unlocked",
		"taken":    "Seat already taken by another user",
		"error":    "Operation failed",
	}[status]
	var res_data *Request
	if status == "taken" || status == "error" {
		res_data = nil
	} else {
		res_data = &body
	}
	return c.JSON(fiber.Map{
		"success": status != "error",
		"message": res_msg,
		"status":  status,
		"data":    res_data,
	})
}

func (ctl *SeatController) GetLockedSeats(c *fiber.Ctx) error {
	showID := c.Query("show_id")
	locked, err := ctl.seatService.GetLockedSeats(context.Background(), showID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get locked seats",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    locked,
		"message": "Locked seats fetched",
	})
}

func StringToUint(s string) (uint, error) {
	u64, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(u64), nil
}
