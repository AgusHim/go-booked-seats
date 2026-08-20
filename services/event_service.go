package services

import (
	"context"
	"go-ticketing/models"
	"go-ticketing/repositories"
)

type EventService interface {
	GetEvents() ([]models.Event, error)
	GetEvent(id string) (*models.Event, error)
	CreateEvent(event *models.Event) error
	UpdateEvent(event *models.Event) error
	DeleteEvent(id string) error
	ListPublished(
		ctx context.Context,
		filter repositories.PublicEventFilter,
	) ([]models.PublicEvent, int64, error)
	FindPublished(ctx context.Context, idOrSlug string) (*models.PublicEvent, error)
	ListPublishedForFollowed(
		ctx context.Context,
		userID string,
		limit int,
	) ([]models.PublicEvent, error)
}

func (s *eventService) ListPublished(
	ctx context.Context,
	filter repositories.PublicEventFilter,
) ([]models.PublicEvent, int64, error) {
	return s.repo.ListPublished(ctx, filter)
}

func (s *eventService) FindPublished(
	ctx context.Context,
	idOrSlug string,
) (*models.PublicEvent, error) {
	return s.repo.FindPublished(ctx, idOrSlug)
}

func (s *eventService) ListPublishedForFollowed(
	ctx context.Context,
	userID string,
	limit int,
) ([]models.PublicEvent, error) {
	return s.repo.ListPublishedForFollowed(ctx, userID, limit)
}

type eventService struct {
	repo repositories.EventRepository
}

func NewEventService(repo repositories.EventRepository) EventService {
	return &eventService{repo}
}

func (s *eventService) GetEvents() ([]models.Event, error) {
	return s.repo.GetEvents()
}

func (s *eventService) GetEvent(id string) (*models.Event, error) {
	return s.repo.GetEvent(id)
}

func (s *eventService) UpdateEvent(event *models.Event) error {
	// Let's first ensure the event exists
	existingEvent, err := s.repo.GetEvent(event.ID)
	if err != nil {
		return err
	}

	// Update mutable fields (for now just WarStartDate and standard fields)
	existingEvent.Name = event.Name
	existingEvent.Location = event.Location
	existingEvent.Description = event.Description
	existingEvent.Status = event.Status
	existingEvent.WarStartDate = event.WarStartDate
	if !event.Date.IsZero() {
		existingEvent.Date = event.Date
	}

	return s.repo.UpdateEvent(existingEvent)
}

func (s *eventService) CreateEvent(event *models.Event) error {
	return s.repo.CreateEvent(event)
}

func (s *eventService) DeleteEvent(id string) error {
	return s.repo.DeleteEvent(id)
}
