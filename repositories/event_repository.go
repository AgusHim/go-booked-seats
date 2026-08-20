package repositories

import (
	"context"
	"go-ticketing/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

type PublicEventFilter struct {
	Query         string
	CommunitySlug string
	Location      string
	DateFrom      *time.Time
	DateTo        *time.Time
	Limit         int
	Offset        int
}

type EventRepository interface {
	GetEvents() ([]models.Event, error)
	GetEvent(id string) (*models.Event, error)
	CreateEvent(event *models.Event) error
	UpdateEvent(event *models.Event) error
	DeleteEvent(id string) error
	ListPublished(
		ctx context.Context,
		filter PublicEventFilter,
	) ([]models.PublicEvent, int64, error)
	FindPublished(ctx context.Context, idOrSlug string) (*models.PublicEvent, error)
	ListPublishedForFollowed(
		ctx context.Context,
		userID string,
		limit int,
	) ([]models.PublicEvent, error)
}

type publicEventRow struct {
	ID            string
	Slug          string
	Name          string
	Date          time.Time
	Location      string
	Description   string
	Status        string
	ImageURL      string
	Color         string
	WarStartDate  *time.Time
	UpdatedAt     time.Time
	CommunityID   string
	CommunitySlug string
	CommunityName string
	CommunityType string
	CommunityLogo string
}

func (r *eventRepository) ListPublished(
	ctx context.Context,
	filter PublicEventFilter,
) ([]models.PublicEvent, int64, error) {
	query := r.publicEventQuery(ctx, filter)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if filter.Limit <= 0 {
		filter.Limit = 12
	}
	if filter.Limit > 50 {
		filter.Limit = 50
	}
	var rows []publicEventRow
	err := r.selectPublicEvents(query).
		Order("events.date ASC").
		Limit(filter.Limit).
		Offset(max(filter.Offset, 0)).
		Scan(&rows).Error
	return mapPublicEventRows(rows), total, err
}

func (r *eventRepository) FindPublished(
	ctx context.Context,
	idOrSlug string,
) (*models.PublicEvent, error) {
	var row publicEventRow
	err := r.selectPublicEvents(r.publicEventQuery(ctx, PublicEventFilter{})).
		Where("events.id = ? OR events.slug = ?", idOrSlug, idOrSlug).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	event := mapPublicEventRow(row)
	return &event, nil
}

func (r *eventRepository) ListPublishedForFollowed(
	ctx context.Context,
	userID string,
	limit int,
) ([]models.PublicEvent, error) {
	if limit <= 0 || limit > 20 {
		limit = 6
	}
	var rows []publicEventRow
	err := r.selectPublicEvents(r.publicEventQuery(ctx, PublicEventFilter{})).
		Joins(
			"JOIN community_follows ON community_follows.community_id = communities.id",
		).
		Where("community_follows.user_id = ?", userID).
		Order("events.date ASC").
		Limit(limit).
		Scan(&rows).Error
	return mapPublicEventRows(rows), err
}

func (r *eventRepository) publicEventQuery(
	ctx context.Context,
	filter PublicEventFilter,
) *gorm.DB {
	query := r.db.WithContext(ctx).
		Table("events").
		Joins(
			"JOIN event_community_assignments ON event_community_assignments.event_id = events.id",
		).
		Joins(
			"JOIN communities ON communities.id = event_community_assignments.community_id",
		).
		Where("events.status = ?", "published").
		Where("communities.status = ?", models.CommunityStatusActive)
	if value := strings.ToLower(strings.TrimSpace(filter.Query)); value != "" {
		like := "%" + value + "%"
		query = query.Where(
			"(LOWER(events.name) LIKE ? OR LOWER(events.description) LIKE ? OR LOWER(events.location) LIKE ?)",
			like,
			like,
			like,
		)
	}
	if value := strings.ToLower(strings.TrimSpace(filter.CommunitySlug)); value != "" {
		query = query.Where("communities.slug = ?", value)
	}
	if value := strings.ToLower(strings.TrimSpace(filter.Location)); value != "" {
		query = query.Where("LOWER(events.location) LIKE ?", "%"+value+"%")
	}
	if filter.DateFrom != nil {
		query = query.Where("events.date >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("events.date <= ?", *filter.DateTo)
	}
	return query
}

func (r *eventRepository) selectPublicEvents(query *gorm.DB) *gorm.DB {
	return query.Select(`
		events.id,
		events.slug,
		events.name,
		events.date,
		events.location,
		events.description,
		events.status,
		events.image_url,
		events.color,
		events.war_start_date,
		events.updated_at,
		communities.id AS community_id,
		communities.slug AS community_slug,
		communities.name AS community_name,
		communities.type AS community_type,
		communities.logo_url AS community_logo
	`)
}

func mapPublicEventRows(rows []publicEventRow) []models.PublicEvent {
	events := make([]models.PublicEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, mapPublicEventRow(row))
	}
	return events
}

func mapPublicEventRow(row publicEventRow) models.PublicEvent {
	return models.PublicEvent{
		ID:           row.ID,
		Slug:         row.Slug,
		Name:         row.Name,
		Date:         row.Date,
		Location:     row.Location,
		Description:  row.Description,
		Status:       row.Status,
		ImageURL:     row.ImageURL,
		Color:        row.Color,
		WarStartDate: row.WarStartDate,
		UpdatedAt:    row.UpdatedAt,
		Community: models.PublicEventCommunity{
			ID:      row.CommunityID,
			Slug:    row.CommunitySlug,
			Name:    row.CommunityName,
			Type:    row.CommunityType,
			LogoURL: row.CommunityLogo,
		},
	}
}

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db}
}

func (r *eventRepository) GetEvents() ([]models.Event, error) {
	var events []models.Event
	err := r.db.Find(&events).Error
	return events, err
}

func (r *eventRepository) GetEvent(id string) (*models.Event, error) {
	var event models.Event
	err := r.db.First(&event, "id = ? OR slug = ?", id, id).Error
	if err == gorm.ErrRecordNotFound {
		// If not found, create default event
		if id == "default" {
			event = models.Event{
				ID:          "default",
				Name:        "Default Event",
				Location:    "Main Venue",
				Description: "Auto generated default event",
				Status:      "active",
			}
			if errCreate := r.db.Create(&event).Error; errCreate != nil {
				return nil, errCreate
			}
			return &event, nil
		}
		return nil, err
	}
	return &event, err
}

func (r *eventRepository) UpdateEvent(event *models.Event) error {
	return r.db.Save(event).Error
}

func (r *eventRepository) CreateEvent(event *models.Event) error {
	return r.db.Create(event).Error
}

func (r *eventRepository) DeleteEvent(id string) error {
	return r.db.Delete(&models.Event{}, "id = ?", id).Error
}
