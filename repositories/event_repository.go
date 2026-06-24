package repositories

import (
	"go-ticketing/models"

	"gorm.io/gorm"
)

type EventRepository interface {
	GetEvents() ([]models.Event, error)
	GetEvent(id string) (*models.Event, error)
	CreateEvent(event *models.Event) error
	UpdateEvent(event *models.Event) error
	DeleteEvent(id string) error
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
	err := r.db.First(&event, "id = ?", id).Error
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
