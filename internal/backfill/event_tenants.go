package backfill

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go-ticketing/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventTenantOptions struct {
	OwnerEmail    string
	CommunityName string
	CommunitySlug string
	CommunityType string
	Apply         bool
}

type EventTenantReport struct {
	CommunityID       string
	WouldCreateTenant bool
	TotalEvents       int64
	AlreadyAssigned   int64
	UnassignedEvents  int64
	AssignedEvents    int64
}

func BackfillEventTenants(
	db *gorm.DB,
	options EventTenantOptions,
) (*EventTenantReport, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	options.OwnerEmail = strings.ToLower(strings.TrimSpace(options.OwnerEmail))
	options.CommunityName = strings.TrimSpace(options.CommunityName)
	options.CommunitySlug = strings.TrimSpace(options.CommunitySlug)
	if options.OwnerEmail == "" || options.CommunityName == "" || options.CommunitySlug == "" {
		return nil, errors.New("owner email, community name, and community slug are required")
	}
	if options.CommunityType == "" {
		options.CommunityType = models.CommunityTypeGeneral
	}
	if !models.IsCommunityType(options.CommunityType) {
		return nil, fmt.Errorf("unsupported community type %q", options.CommunityType)
	}
	for _, table := range []interface{}{
		&models.User{},
		&models.Event{},
		&models.Community{},
		&models.CommunityMember{},
		&models.EventCommunityAssignment{},
	} {
		if !db.Migrator().HasTable(table) {
			return nil, fmt.Errorf("required table for %T is missing; run migrations first", table)
		}
	}

	var owner models.User
	if err := db.Where("LOWER(email) = ?", options.OwnerEmail).First(&owner).Error; err != nil {
		return nil, fmt.Errorf("find explicit backfill owner: %w", err)
	}

	report := &EventTenantReport{}
	if err := db.Model(&models.Event{}).Count(&report.TotalEvents).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.EventCommunityAssignment{}).
		Count(&report.AlreadyAssigned).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&models.Event{}).
		Where("id NOT IN (?)", db.Model(&models.EventCommunityAssignment{}).Select("event_id")).
		Count(&report.UnassignedEvents).Error; err != nil {
		return nil, err
	}

	var community models.Community
	err := db.Where("slug = ?", options.CommunitySlug).First(&community).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		report.WouldCreateTenant = true
		community = models.Community{
			Name:   options.CommunityName,
			Slug:   options.CommunitySlug,
			Type:   options.CommunityType,
			Status: models.CommunityStatusActive,
		}
	case err != nil:
		return nil, err
	case community.Name != options.CommunityName || community.Type != options.CommunityType:
		return nil, errors.New("existing community does not match requested name and type")
	}
	report.CommunityID = community.ID
	if !options.Apply {
		return report, nil
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if report.WouldCreateTenant {
			if err := tx.Create(&community).Error; err != nil {
				return err
			}
		}
		report.CommunityID = community.ID

		var member models.CommunityMember
		err := tx.Where(
			"community_id = ? AND user_id = ?",
			community.ID,
			owner.ID,
		).First(&member).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			member = models.CommunityMember{
				CommunityID: community.ID,
				UserID:      owner.ID,
				Role:        models.CommunityRoleOwner,
				Status:      models.CommunityMemberStatusActive,
			}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else if member.Role != models.CommunityRoleOwner ||
			member.Status != models.CommunityMemberStatusActive {
			return errors.New("explicit owner has a conflicting existing membership")
		}

		var eventIDs []string
		if err := tx.Model(&models.Event{}).
			Where("id NOT IN (?)", tx.Model(&models.EventCommunityAssignment{}).Select("event_id")).
			Pluck("id", &eventIDs).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		for _, eventID := range eventIDs {
			assignment := &models.EventCommunityAssignment{
				EventID:     eventID,
				CommunityID: community.ID,
				Source:      models.EventTenantSourceLegacyBackfill,
				AssignedAt:  now,
			}
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(assignment)
			if result.Error != nil {
				return result.Error
			}
			report.AssignedEvents += result.RowsAffected
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}
