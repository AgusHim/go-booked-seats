package middleware

import (
	"go-ticketing/models"
	"go-ticketing/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthProtected(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := utils.ParseJWT(c.Get("Authorization"))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Missing token"})
		}

		userID, ok := utils.ClaimString(claims, "user_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid token"})
		}

		c.Locals("user_id", userID)
		return c.Next()
	}
}

func AdminProtected(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := utils.ParseJWT(c.Get("Authorization"))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
		}

		userID, ok := utils.ClaimString(claims, "user_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid token"})
		}

		var user models.User
		if err := db.First(&user, "id = ?", userID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid user"})
		}
		if user.Role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Admin access required"})
		}

		c.Locals("user_id", user.ID)
		c.Locals("role", user.Role)
		return c.Next()
	}
}

func TicketProtected(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := utils.ParseJWT(c.Get("Authorization"))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
		}

		tokenType, ok := utils.ClaimString(claims, "type")
		if !ok || tokenType != "war_kursi" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid ticket token"})
		}

		ticketID, ok := utils.ClaimString(claims, "ticket_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid ticket token"})
		}

		var ticket models.Ticket
		if err := db.First(&ticket, "id = ?", ticketID).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Ticket not found"})
		}

		c.Locals("ticket_id", ticket.ID)
		c.Locals("ticket", &ticket)
		return c.Next()
	}
}
