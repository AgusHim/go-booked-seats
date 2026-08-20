package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID       string   `json:"id" gorm:"primaryKey;type:uuid"`
	Name     string   `json:"name"`
	Email    string   `json:"email" gorm:"uniqueIndex"`
	Role     string   `json:"role"`
	Password string   `json:"-" gorm:"type:text"`
	Tickets  []Ticket `json:"tickets,omitempty" gorm:"foreignKey:UserID"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	if u.Role == "" {
		u.Role = "user"
	}
	return nil
}
