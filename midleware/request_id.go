package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		requestID := strings.TrimSpace(ctx.Get(RequestIDHeader))
		if _, err := uuid.Parse(requestID); err != nil {
			requestID = uuid.NewString()
		}

		ctx.Locals("request_id", requestID)
		ctx.Set(RequestIDHeader, requestID)
		return ctx.Next()
	}
}
