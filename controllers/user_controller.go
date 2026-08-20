package controllers

import (
	"errors"
	"go-ticketing/httpapi"
	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/services"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

const refreshCookieName = "usloop_refresh"

type UserController struct {
	userService services.UserService
}

func NewUserController(userService services.UserService) *UserController {
	return &UserController{userService}
}

func (ctl *UserController) Register(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}

	verificationToken, err := ctl.userService.Register(c.UserContext(), &user)
	if err != nil {
		return httpapi.Error(c, fiber.StatusBadRequest, "REGISTRATION_FAILED", err.Error())
	}

	data := fiber.Map{"message": "User registered successfully"}
	addDevelopmentToken(data, verificationToken)
	return httpapi.Data(c, fiber.StatusCreated, data)
}

func (ctl *UserController) Login(c *fiber.Ctx) error {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}

	result, err := ctl.userService.Login(
		c.UserContext(),
		payload.Email,
		payload.Password,
		sessionMetadata(c),
	)
	if err != nil {
		return httpapi.Error(
			c,
			fiber.StatusUnauthorized,
			"INVALID_CREDENTIALS",
			"Invalid email or password",
		)
	}

	setRefreshCookie(c, result.RefreshToken, result.RefreshExpiresAt)
	return c.JSON(fiber.Map{
		"token":      result.AccessToken,
		"expires_at": result.AccessExpiresAt,
		"data":       result.User,
		"meta":       fiber.Map{"request_id": httpapi.RequestID(c)},
	})
}

func (ctl *UserController) Refresh(c *fiber.Ctx) error {
	result, err := ctl.userService.Refresh(
		c.UserContext(),
		c.Cookies(refreshCookieName),
		sessionMetadata(c),
	)
	if err != nil {
		clearRefreshCookie(c)
		return httpapi.Error(
			c,
			fiber.StatusUnauthorized,
			"REFRESH_SESSION_INVALID",
			"Session is invalid or expired",
		)
	}

	setRefreshCookie(c, result.RefreshToken, result.RefreshExpiresAt)
	return c.JSON(fiber.Map{
		"token":      result.AccessToken,
		"expires_at": result.AccessExpiresAt,
		"data":       result.User,
		"meta":       fiber.Map{"request_id": httpapi.RequestID(c)},
	})
}

func (ctl *UserController) Logout(c *fiber.Ctx) error {
	if err := ctl.userService.Logout(c.UserContext(), c.Cookies(refreshCookieName)); err != nil {
		return httpapi.Error(c, fiber.StatusInternalServerError, "LOGOUT_FAILED", "Unable to end session")
	}
	clearRefreshCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

func (ctl *UserController) RequestEmailVerification(c *fiber.Ctx) error {
	var payload struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return httpapi.Error(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}
	token, err := ctl.userService.RequestEmailVerification(c.UserContext(), payload.Email)
	if err != nil {
		return httpapi.Error(c, fiber.StatusInternalServerError, "REQUEST_FAILED", "Unable to process request")
	}
	data := fiber.Map{"message": "If the account exists, verification instructions will be sent"}
	addDevelopmentToken(data, token)
	return httpapi.Data(c, fiber.StatusAccepted, data)
}

func (ctl *UserController) VerifyEmail(c *fiber.Ctx) error {
	var payload struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return httpapi.Error(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}
	if err := ctl.userService.VerifyEmail(c.UserContext(), payload.Token); err != nil {
		return authTokenError(c, err)
	}
	return httpapi.Data(c, fiber.StatusOK, fiber.Map{"message": "Email verified"})
}

func (ctl *UserController) RequestPasswordReset(c *fiber.Ctx) error {
	var payload struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return httpapi.Error(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}
	token, err := ctl.userService.RequestPasswordReset(c.UserContext(), payload.Email)
	if err != nil {
		return httpapi.Error(c, fiber.StatusInternalServerError, "REQUEST_FAILED", "Unable to process request")
	}
	data := fiber.Map{"message": "If the account exists, reset instructions will be sent"}
	addDevelopmentToken(data, token)
	return httpapi.Data(c, fiber.StatusAccepted, data)
}

func (ctl *UserController) ResetPassword(c *fiber.Ctx) error {
	var payload struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return httpapi.Error(c, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}
	if err := ctl.userService.ResetPassword(c.UserContext(), payload.Token, payload.Password); err != nil {
		if errors.Is(err, repositories.ErrAuthTokenUnavailable) {
			return authTokenError(c, err)
		}
		return httpapi.Error(c, fiber.StatusBadRequest, "INVALID_PASSWORD", err.Error())
	}
	clearRefreshCookie(c)
	return httpapi.Data(c, fiber.StatusOK, fiber.Map{"message": "Password updated"})
}

func (ctl *UserController) ListSessions(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil {
		return httpapi.Error(c, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	sessions, err := ctl.userService.ListSessions(c.UserContext(), user.ID)
	if err != nil {
		return httpapi.Error(c, fiber.StatusInternalServerError, "SESSIONS_FAILED", "Unable to load sessions")
	}
	currentID, _ := c.Locals("session_id").(string)
	data := make([]fiber.Map, 0, len(sessions))
	for _, session := range sessions {
		data = append(data, fiber.Map{
			"id":           session.ID,
			"user_agent":   session.UserAgent,
			"ip_address":   session.IPAddress,
			"expires_at":   session.ExpiresAt,
			"last_used_at": session.LastUsedAt,
			"created_at":   session.CreatedAt,
			"current":      session.ID == currentID,
		})
	}
	return httpapi.Data(c, fiber.StatusOK, data)
}

func (ctl *UserController) RevokeSession(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil {
		return httpapi.Error(c, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	if err := ctl.userService.RevokeSession(
		c.UserContext(),
		user.ID,
		c.Params("session_id"),
	); err != nil {
		if errors.Is(err, repositories.ErrAuthTokenUnavailable) {
			return httpapi.Error(c, fiber.StatusNotFound, "SESSION_NOT_FOUND", "Session not found")
		}
		return httpapi.Error(c, fiber.StatusInternalServerError, "SESSION_REVOKE_FAILED", "Unable to revoke session")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (ctl *UserController) Me(c *fiber.Ctx) error {
	user, ok := c.Locals("user").(*models.User)
	if !ok || user == nil {
		return httpapi.Error(c, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	return httpapi.Data(c, fiber.StatusOK, user)
}

func (ctl *UserController) FindAll(c *fiber.Ctx) error {
	users, err := ctl.userService.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(users)
}

func (ctl *UserController) FindByID(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := ctl.userService.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "User not found"})
	}
	return c.JSON(user)
}

func (ctl *UserController) Create(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}
	if err := ctl.userService.Create(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(user)
}

func (ctl *UserController) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid input"})
	}
	if err := ctl.userService.Update(id, &user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "User updated"})
}

func (ctl *UserController) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := ctl.userService.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "User deleted"})
}

func sessionMetadata(c *fiber.Ctx) services.SessionMetadata {
	return services.SessionMetadata{
		UserAgent: c.Get(fiber.HeaderUserAgent),
		IPAddress: c.IP(),
	}
}

func setRefreshCookie(c *fiber.Ctx, token string, expiresAt time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		Secure:   refreshCookieSecure(),
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func clearRefreshCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   refreshCookieSecure(),
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func refreshCookieSecure() bool {
	return strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true") ||
		strings.EqualFold(os.Getenv("APP_ENV"), "production")
}

func addDevelopmentToken(data fiber.Map, token string) {
	if token != "" && strings.EqualFold(os.Getenv("AUTH_TOKEN_DELIVERY"), "development_response") {
		data["development_token"] = token
	}
}

func authTokenError(c *fiber.Ctx, err error) error {
	if errors.Is(err, repositories.ErrAuthTokenUnavailable) {
		return httpapi.Error(
			c,
			fiber.StatusBadRequest,
			"AUTH_TOKEN_INVALID",
			"Token is invalid, expired, or already used",
		)
	}
	return httpapi.Error(c, fiber.StatusInternalServerError, "REQUEST_FAILED", "Unable to process request")
}
