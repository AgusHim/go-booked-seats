package controllers

import (
	"errors"
	"go-ticketing/httpapi"
	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/services"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type EventController struct {
	service services.EventService
}

func (c *EventController) ListPublic(ctx *fiber.Ctx) error {
	filter, page, limit, err := publicEventFilter(ctx)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_FILTER", err.Error())
	}
	events, total, err := c.service.ListPublished(ctx.Context(), filter)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "EVENT_LIST_FAILED", "Unable to load events")
	}
	return ctx.JSON(fiber.Map{
		"data": events,
		"meta": fiber.Map{
			"request_id": httpapi.RequestID(ctx),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

func (c *EventController) ListCommunityPublic(ctx *fiber.Ctx) error {
	filter, page, limit, err := publicEventFilter(ctx)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_FILTER", err.Error())
	}
	filter.CommunitySlug = ctx.Params("slug")
	events, total, err := c.service.ListPublished(ctx.Context(), filter)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "EVENT_LIST_FAILED", "Unable to load events")
	}
	return ctx.JSON(fiber.Map{
		"data": events,
		"meta": fiber.Map{
			"request_id": httpapi.RequestID(ctx),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

func (c *EventController) GetPublic(ctx *fiber.Ctx) error {
	event, err := c.service.FindPublished(ctx.Context(), ctx.Params("id_or_slug"))
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = fiber.StatusNotFound
		}
		return httpapi.Error(ctx, status, "EVENT_NOT_FOUND", "Event not found")
	}
	return httpapi.Data(ctx, fiber.StatusOK, event)
}

func (c *EventController) ListFollowingFeed(ctx *fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	limit, _ := strconv.Atoi(ctx.Query("limit", "6"))
	events, err := c.service.ListPublishedForFollowed(ctx.Context(), userID, limit)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "EVENT_FEED_FAILED", "Unable to load events")
	}
	return httpapi.Data(ctx, fiber.StatusOK, events)
}

func publicEventFilter(
	ctx *fiber.Ctx,
) (repositories.PublicEventFilter, int, int, error) {
	page, err := strconv.Atoi(ctx.Query("page", "1"))
	if err != nil || page < 1 {
		return repositories.PublicEventFilter{}, 0, 0, errors.New("page must be a positive integer")
	}
	limit, err := strconv.Atoi(ctx.Query("limit", "12"))
	if err != nil || limit < 1 || limit > 50 {
		return repositories.PublicEventFilter{}, 0, 0, errors.New("limit must be between 1 and 50")
	}
	filter := repositories.PublicEventFilter{
		Query:         strings.TrimSpace(ctx.Query("q")),
		CommunitySlug: strings.TrimSpace(ctx.Query("community")),
		Location:      strings.TrimSpace(ctx.Query("location")),
		Limit:         limit,
		Offset:        (page - 1) * limit,
	}
	if value := strings.TrimSpace(ctx.Query("date_from")); value != "" {
		date, err := time.Parse("2006-01-02", value)
		if err != nil {
			return filter, 0, 0, errors.New("date_from must use YYYY-MM-DD")
		}
		filter.DateFrom = &date
	}
	if value := strings.TrimSpace(ctx.Query("date_to")); value != "" {
		date, err := time.Parse("2006-01-02", value)
		if err != nil {
			return filter, 0, 0, errors.New("date_to must use YYYY-MM-DD")
		}
		endOfDay := date.Add(24*time.Hour - time.Nanosecond)
		filter.DateTo = &endOfDay
	}
	return filter, page, limit, nil
}

func NewEventController(service services.EventService) *EventController {
	return &EventController{service}
}

func (c *EventController) GetEvents(ctx *fiber.Ctx) error {
	events, err := c.service.GetEvents()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": events, "message": "Success get events"})
}

func (c *EventController) GetEvent(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	event, err := c.service.GetEvent(id)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "Event not found"})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": event, "message": "Success get event"})
}

func (c *EventController) UpdateEvent(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	var event models.Event
	if err := ctx.BodyParser(&event); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	event.ID = id
	if err := c.service.UpdateEvent(&event); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "data": event, "message": "Event updated"})
}

func (c *EventController) CreateEvent(ctx *fiber.Ctx) error {
	var event models.Event
	if err := ctx.BodyParser(&event); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	if err := c.service.CreateEvent(&event); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": event, "message": "Event created"})
}

func (c *EventController) DeleteEvent(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.service.DeleteEvent(id); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": err.Error()})
	}

	return ctx.JSON(fiber.Map{"success": true, "message": "Event deleted"})
}
