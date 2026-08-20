package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	CommunityTypeGeneral = "general"
	CommunityTypeDakwah  = "dakwah"
	CommunityTypeRunning = "running"

	CommunityStatusActive   = "active"
	CommunityStatusInactive = "inactive"

	CommunityMemberStatusActive  = "active"
	CommunityMemberStatusInvited = "invited"
	CommunityMemberStatusRemoved = "removed"

	CommunityRoleOwner        = "owner"
	CommunityRoleAdmin        = "admin"
	CommunityRoleEventManager = "event_manager"
	CommunityRoleCheckinStaff = "checkin_staff"
	CommunityRoleModerator    = "moderator"
	CommunityRoleMentor       = "mentor"
)

type Community struct {
	ID          string            `json:"id" gorm:"primaryKey;type:uuid"`
	Slug        string            `json:"slug" gorm:"uniqueIndex;size:160;not null"`
	Name        string            `json:"name" gorm:"size:160;not null"`
	Description string            `json:"description" gorm:"type:text"`
	Type        string            `json:"type" gorm:"size:32;not null;default:'general'"`
	Status      string            `json:"status" gorm:"size:32;not null;default:'active'"`
	LogoURL     string            `json:"logo_url" gorm:"type:text"`
	CoverURL    string            `json:"cover_url" gorm:"type:text"`
	Location    string            `json:"location" gorm:"size:255"`
	Members     []CommunityMember `json:"-" gorm:"foreignKey:CommunityID"`
	CreatedAt   time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}

func (c *Community) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.Name = strings.TrimSpace(c.Name)
	c.Slug = generateSlug(c.Slug)
	if c.Slug == "" {
		c.Slug = generateSlug(c.Name)
	}
	if c.Type == "" {
		c.Type = CommunityTypeGeneral
	}
	if c.Status == "" {
		c.Status = CommunityStatusActive
	}
	return nil
}

type CommunityMember struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid"`
	CommunityID string    `json:"community_id" gorm:"type:uuid;not null;uniqueIndex:idx_community_user"`
	Community   Community `json:"-" gorm:"foreignKey:CommunityID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	UserID      string    `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_community_user"`
	User        User      `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Role        string    `json:"role" gorm:"size:32;not null"`
	Status      string    `json:"status" gorm:"size:32;not null;default:'active'"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (m *CommunityMember) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.Status == "" {
		m.Status = CommunityMemberStatusActive
	}
	return nil
}

type CommunityInvitation struct {
	ID          string     `json:"id" gorm:"primaryKey;type:uuid"`
	CommunityID string     `json:"community_id" gorm:"type:uuid;not null;index"`
	Community   Community  `json:"-" gorm:"foreignKey:CommunityID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Email       string     `json:"email" gorm:"size:320;not null;index"`
	Role        string     `json:"role" gorm:"size:32;not null"`
	TokenHash   string     `json:"-" gorm:"size:64;not null;uniqueIndex"`
	InvitedBy   string     `json:"invited_by" gorm:"type:uuid;not null"`
	Inviter     User       `json:"-" gorm:"foreignKey:InvitedBy"`
	ExpiresAt   time.Time  `json:"expires_at" gorm:"not null;index"`
	AcceptedAt  *time.Time `json:"accepted_at"`
	AcceptedBy  *string    `json:"accepted_by" gorm:"type:uuid"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

type CommunityFollow struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid"`
	CommunityID string    `json:"community_id" gorm:"type:uuid;not null;uniqueIndex:idx_community_follower"`
	UserID      string    `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_community_follower;index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (follow *CommunityFollow) BeforeCreate(tx *gorm.DB) error {
	if follow.ID == "" {
		follow.ID = uuid.NewString()
	}
	return nil
}

func (i *CommunityInvitation) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	i.Email = strings.ToLower(strings.TrimSpace(i.Email))
	return nil
}

func IsCommunityType(value string) bool {
	switch value {
	case CommunityTypeGeneral, CommunityTypeDakwah, CommunityTypeRunning:
		return true
	default:
		return false
	}
}

func IsCommunityRole(value string) bool {
	switch value {
	case CommunityRoleOwner,
		CommunityRoleAdmin,
		CommunityRoleEventManager,
		CommunityRoleCheckinStaff,
		CommunityRoleModerator,
		CommunityRoleMentor:
		return true
	default:
		return false
	}
}
