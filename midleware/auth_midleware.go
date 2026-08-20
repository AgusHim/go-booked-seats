package middleware

import (
	"errors"
	"go-ticketing/authorization"
	"go-ticketing/httpapi"
	"go-ticketing/models"
	"go-ticketing/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func AuthProtected(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, db)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
		}
		c.Locals("user_id", user.ID)
		c.Locals("user", user)
		c.Locals("role", user.Role)
		if claims, parseErr := utils.ParseJWT(c.Get("Authorization")); parseErr == nil {
			if sessionID, ok := utils.ClaimString(claims, "sid"); ok {
				c.Locals("session_id", sessionID)
			}
		}
		return c.Next()
	}
}

func AdminProtected(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, err := authenticateUser(c, db)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid or expired token"})
		}
		if user.Role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Admin access required"})
		}

		c.Locals("user_id", user.ID)
		c.Locals("user", user)
		c.Locals("role", user.Role)
		return c.Next()
	}
}

func CommunityPermission(
	db *gorm.DB,
	permission authorization.Permission,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(*models.User)
		if !ok || user == nil {
			return httpapi.Error(c, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
		}

		communityID := c.Params("community_id")
		var community models.Community
		if err := db.First(&community, "id = ?", communityID).Error; err != nil {
			status := fiber.StatusInternalServerError
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status = fiber.StatusNotFound
			}
			return httpapi.Error(c, status, "COMMUNITY_NOT_FOUND", "Community not found")
		}

		if user.Role == "admin" {
			c.Locals("community", &community)
			c.Locals("community_id", community.ID)
			c.Locals("community_role", "platform_admin")
			return c.Next()
		}

		var member models.CommunityMember
		err := db.Where(
			"community_id = ? AND user_id = ? AND status = ?",
			community.ID,
			user.ID,
			models.CommunityMemberStatusActive,
		).First(&member).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpapi.Error(c, fiber.StatusForbidden, "COMMUNITY_ACCESS_DENIED", "Community access denied")
		}
		if err != nil {
			return httpapi.Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		if !authorization.HasPermission(member.Role, permission) {
			return httpapi.Error(c, fiber.StatusForbidden, "COMMUNITY_PERMISSION_REQUIRED", "Community permission required")
		}

		c.Locals("community", &community)
		c.Locals("community_id", community.ID)
		c.Locals("community_member", &member)
		c.Locals("community_role", member.Role)
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

func authenticateUser(c *fiber.Ctx, db *gorm.DB) (*models.User, error) {
	claims, err := utils.ParseJWT(c.Get("Authorization"))
	if err != nil {
		return nil, err
	}

	tokenType, ok := utils.ClaimString(claims, "type")
	if !ok || (tokenType != "user_session" && tokenType != "admin") {
		return nil, errors.New("invalid user token")
	}

	userID, ok := utils.ClaimString(claims, "user_id")
	if !ok {
		return nil, errors.New("invalid user token")
	}

	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
