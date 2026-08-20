package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AuthTokenPurposeEmailVerification = "email_verification"
	AuthTokenPurposePasswordReset     = "password_reset"
)

type AuthSession struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid"`
	UserID       string     `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash    string     `json:"-" gorm:"size:64;not null;uniqueIndex"`
	UserAgent    string     `json:"user_agent" gorm:"size:512"`
	IPAddress    string     `json:"ip_address" gorm:"size:64"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"not null;index"`
	LastUsedAt   time.Time  `json:"last_used_at" gorm:"not null"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" gorm:"index"`
	ReplacedByID *string    `json:"replaced_by_id,omitempty" gorm:"type:uuid"`
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	User         *User      `json:"-" gorm:"foreignKey:UserID;references:ID"`
}

func (session *AuthSession) BeforeCreate(tx *gorm.DB) error {
	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	return nil
}

type AuthToken struct {
	ID        string     `json:"id" gorm:"primaryKey;type:uuid"`
	UserID    string     `json:"user_id" gorm:"type:uuid;not null;index"`
	Purpose   string     `json:"purpose" gorm:"size:32;not null;index"`
	TokenHash string     `json:"-" gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null;index"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

func (token *AuthToken) BeforeCreate(tx *gorm.DB) error {
	if token.ID == "" {
		token.ID = uuid.NewString()
	}
	return nil
}

type UserEmailVerification struct {
	UserID     string     `json:"user_id" gorm:"primaryKey;type:uuid"`
	Email      string     `json:"email" gorm:"size:320;not null;uniqueIndex"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}
