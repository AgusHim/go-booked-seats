package models

import "time"

type PublicEventCommunity struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	LogoURL string `json:"logo_url,omitempty"`
}

type PublicEvent struct {
	ID           string               `json:"id"`
	Slug         string               `json:"slug"`
	Name         string               `json:"name"`
	Date         time.Time            `json:"date"`
	Location     string               `json:"location"`
	Description  string               `json:"description"`
	Status       string               `json:"status"`
	ImageURL     string               `json:"image_url,omitempty"`
	Color        string               `json:"color,omitempty"`
	WarStartDate *time.Time           `json:"war_start_date,omitempty"`
	UpdatedAt    time.Time            `json:"updated_at"`
	Community    PublicEventCommunity `json:"community"`
}
