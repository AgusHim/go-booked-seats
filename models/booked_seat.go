package models

import "time"

type BookedSeat struct {
	ID       string    `json:"id" gorm:"primaryKey"`
	SeatID   string    `json:"seat_id"`
	ShowID   string    `json:"show_id"`
	AdminID  string    `json:"admin_id"`
	Name     string    `json:"name"`
	TicketID string    `json:"ticket_id"`
	CreateAt time.Time `json:"create_at" gorm:"autoCreateTime"`
	UpdateAt time.Time `json:"update_at" gorm:"autoUpdateTime"`
}
