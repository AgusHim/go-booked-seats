package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid"`
	Name        string    `json:"name" validate:"required"`
	Date        time.Time `json:"date" validate:"required"`
	Location    string    `json:"location" validate:"required"`
	Description string    `json:"description" gorm:"type:text"`
	Status      string    `json:"status" validate:"required"`
	ImageURL    string    `json:"image_url" gorm:"type:text"`
	Color       string    `json:"color"`
	Seats       []Seat    `json:"seats,omitempty" gorm:"foreignKey:EventID"`
	Tickets      []Ticket   `json:"tickets,omitempty" gorm:"foreignKey:EventID"`
	WarStartDate *time.Time `json:"war_start_date"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (e *Event) BeforeCreate(tx *gorm.DB) (err error) {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return
}
