package models

import "time"

const EventTenantSourceLegacyBackfill = "legacy_backfill"

// EventCommunityAssignment isolates legacy event ownership without making a
// nullable community_id column authoritative before the existing data has been
// reviewed and backfilled.
type EventCommunityAssignment struct {
	EventID     string     `json:"event_id" gorm:"primaryKey;type:uuid"`
	CommunityID string     `json:"community_id" gorm:"type:uuid;not null;index"`
	Source      string     `json:"source" gorm:"size:32;not null"`
	AssignedAt  time.Time  `json:"assigned_at" gorm:"not null"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	Event       *Event     `json:"-" gorm:"foreignKey:EventID;references:ID"`
	Community   *Community `json:"-" gorm:"foreignKey:CommunityID;references:ID"`
}
