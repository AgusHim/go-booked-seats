package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestRequestIDCreatesAndPreservesValidIDs(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.SendString(ctx.Locals("request_id").(string))
	})

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("request without id: %v", err)
	}
	generated := response.Header.Get(RequestIDHeader)
	if _, err := uuid.Parse(generated); err != nil {
		t.Fatalf("generated request id is not UUID: %q", generated)
	}

	provided := uuid.NewString()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(RequestIDHeader, provided)
	response, err = app.Test(request)
	if err != nil {
		t.Fatalf("request with id: %v", err)
	}
	if got := response.Header.Get(RequestIDHeader); got != provided {
		t.Fatalf("request id=%q want=%q", got, provided)
	}
}
