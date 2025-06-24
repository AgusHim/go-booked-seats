package models

import "time"

type Seat struct {
	ID       string    `json:"id" gorm:"primaryKey" validate:"required"`
	Color    string    `json:"color" validate:"required"`
	Name     string    `json:"name"`
	Category string    `json:"category" validate:"required"`
	CreateAt time.Time `json:"create_at" gorm:"autoCreateTime"`
	UpdateAt time.Time `json:"update_at" gorm:"autoUpdateTime"`
}
