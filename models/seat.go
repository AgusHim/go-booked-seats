package models

import "time"

type Seat struct {
	ID       string    `json:"id" gorm:"primaryKey"`
	Position string    `json:"position" validate:"required"`
	Color    string    `json:"color" validate:"required"`
	Name     string    `json:"name"`
	Gender   string    `json:"gender" gorm:"default:'both'"`
	Category string    `json:"category" validate:"required"`
	EventID  string    `json:"event_id" validate:"required"`
	Event    *Event    `json:"event,omitempty" gorm:"foreignKey:EventID;references:ID"`
	X        float64   `json:"x"`
	Y        float64   `json:"y"`
	Rotation float64   `json:"rotation"`
	Width    float64   `json:"width"`
	Height   float64   `json:"height"`
	CreateAt time.Time `json:"create_at" gorm:"autoCreateTime"`
	UpdateAt time.Time `json:"update_at" gorm:"autoUpdateTime"`
}
