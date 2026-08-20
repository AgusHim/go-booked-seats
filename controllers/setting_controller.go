package controllers

import (
	"go-ticketing/services"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type SettingController struct {
	service services.SettingService
}

func NewSettingController(service services.SettingService) *SettingController {
	return &SettingController{service: service}
}

func (c *SettingController) GetDarisini(ctx *fiber.Ctx) error {
	cookie, err := c.service.GetDarisiniCookie()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"configured": strings.TrimSpace(cookie) != ""},
		"message": "Success get setting",
	})
}

func (c *SettingController) UpdateDarisini(ctx *fiber.Ctx) error {
	var body struct {
		Cookie string `json:"cookie"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	body.Cookie = strings.TrimSpace(body.Cookie)
	if body.Cookie == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Darisini cookie is required",
		})
	}
	if len(body.Cookie) > 16*1024 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Darisini cookie is too large",
		})
	}
	if err := c.service.UpdateDarisiniCookie(body.Cookie); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"configured": true},
		"message": "Darisini cookie updated",
	})
}
