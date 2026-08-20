package httpapi

import "github.com/gofiber/fiber/v2"

type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

func Data(ctx *fiber.Ctx, status int, data interface{}) error {
	return ctx.Status(status).JSON(fiber.Map{
		"data": data,
		"meta": fiber.Map{"request_id": RequestID(ctx)},
	})
}

func Error(
	ctx *fiber.Ctx,
	status int,
	code string,
	message string,
) error {
	return ErrorWithFields(ctx, status, code, message, nil)
}

func ErrorWithFields(
	ctx *fiber.Ctx,
	status int,
	code string,
	message string,
	fields map[string]interface{},
) error {
	return ctx.Status(status).JSON(fiber.Map{
		"error": ErrorDetail{
			Code:    code,
			Message: message,
			Fields:  fields,
		},
		"meta": fiber.Map{"request_id": RequestID(ctx)},
	})
}

func RequestID(ctx *fiber.Ctx) string {
	requestID, _ := ctx.Locals("request_id").(string)
	return requestID
}
