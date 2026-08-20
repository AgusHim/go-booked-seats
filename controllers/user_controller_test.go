package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	middleware "go-ticketing/midleware"
	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserControllerRefreshCookieRotation(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-24-characters")
	t.Setenv("COOKIE_SECURE", "false")

	db, err := gorm.Open(
		sqlite.Open("file:controller-auth?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.AuthSession{},
		&models.AuthToken{},
		&models.UserEmailVerification{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	userService := services.NewUserService(
		repositories.NewUserRepository(db),
		repositories.NewAuthRepository(db),
	)
	if _, err := userService.Register(context.Background(), &models.User{
		Name:     "Owner",
		Email:    "owner@example.test",
		Password: "password-aman",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	controller := NewUserController(userService)
	app := fiber.New()
	app.Use(middleware.RequestID())
	app.Post("/login", controller.Login)
	app.Post("/refresh", controller.Refresh)
	app.Post("/logout", controller.Logout)

	loginResponse := performAuthRequest(
		t,
		app,
		"/login",
		`{"email":"owner@example.test","password":"password-aman"}`,
		nil,
	)
	if loginResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("login status: %d", loginResponse.StatusCode)
	}
	firstCookie := findRefreshCookie(t, loginResponse)
	if !firstCookie.HttpOnly || firstCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("refresh cookie must be HttpOnly and SameSite=Lax: %#v", firstCookie)
	}

	refreshResponse := performAuthRequest(t, app, "/refresh", `{}`, firstCookie)
	if refreshResponse.StatusCode != fiber.StatusOK {
		t.Fatalf("refresh status: %d", refreshResponse.StatusCode)
	}
	secondCookie := findRefreshCookie(t, refreshResponse)
	if secondCookie.Value == firstCookie.Value {
		t.Fatal("refresh endpoint must rotate the cookie value")
	}

	replayResponse := performAuthRequest(t, app, "/refresh", `{}`, firstCookie)
	if replayResponse.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("replayed cookie status: %d", replayResponse.StatusCode)
	}

	logoutResponse := performAuthRequest(t, app, "/logout", `{}`, secondCookie)
	if logoutResponse.StatusCode != fiber.StatusNoContent {
		t.Fatalf("logout status: %d", logoutResponse.StatusCode)
	}
}

func performAuthRequest(
	t *testing.T,
	app *fiber.App,
	path string,
	body string,
	cookie *http.Cookie,
) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	return response
}

func findRefreshCookie(t *testing.T, response *http.Response) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == refreshCookieName && strings.TrimSpace(cookie.Value) != "" {
			return cookie
		}
	}
	var payload map[string]interface{}
	_ = json.NewDecoder(response.Body).Decode(&payload)
	t.Fatalf("refresh cookie missing; response=%#v", payload)
	return nil
}
