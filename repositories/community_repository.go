package repositories

import (
	"context"
	"errors"
	"time"

	"go-ticketing/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrCommunityMemberNotFound = errors.New("community member not found")
	ErrInvitationUnavailable   = errors.New("invitation is expired, accepted, or unavailable")
	ErrAlreadyCommunityMember  = errors.New("user is already an active community member")
)

type CommunityRepository interface {
	CreateWithOwner(
		ctx context.Context,
		community *models.Community,
		owner *models.CommunityMember,
	) error
	FindByID(ctx context.Context, id string) (*models.Community, error)
	FindBySlug(ctx context.Context, slug string) (*models.Community, error)
	ListPublic(ctx context.Context, limit, offset int) ([]models.Community, int64, error)
	ListForUser(ctx context.Context, userID string) ([]models.Community, error)
	FindActiveMember(
		ctx context.Context,
		communityID string,
		userID string,
	) (*models.CommunityMember, error)
	ListMembers(ctx context.Context, communityID string) ([]models.CommunityMember, error)
	CreateInvitation(ctx context.Context, invitation *models.CommunityInvitation) error
	FindInvitationByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (*models.CommunityInvitation, error)
	AcceptInvitation(
		ctx context.Context,
		invitation *models.CommunityInvitation,
		userID string,
	) (*models.CommunityMember, error)
	Follow(ctx context.Context, communityID, userID string) error
	Unfollow(ctx context.Context, communityID, userID string) error
	IsFollowing(ctx context.Context, communityID, userID string) (bool, error)
	CountFollowers(ctx context.Context, communityID string) (int64, error)
	ListFollowing(ctx context.Context, userID string) ([]models.Community, error)
	UpdateProfile(ctx context.Context, community *models.Community) error
	FindMemberByID(
		ctx context.Context,
		communityID string,
		memberID string,
	) (*models.CommunityMember, error)
	UpdateMemberRole(ctx context.Context, memberID, role string) error
	RemoveMember(ctx context.Context, memberID string) error
}

func (r *communityRepository) UpdateProfile(
	ctx context.Context,
	community *models.Community,
) error {
	return r.db.WithContext(ctx).
		Model(&models.Community{}).
		Where("id = ?", community.ID).
		Updates(map[string]interface{}{
			"name":        community.Name,
			"description": community.Description,
			"logo_url":    community.LogoURL,
			"cover_url":   community.CoverURL,
			"location":    community.Location,
			"updated_at":  time.Now().UTC(),
		}).Error
}

func (r *communityRepository) FindMemberByID(
	ctx context.Context,
	communityID string,
	memberID string,
) (*models.CommunityMember, error) {
	var member models.CommunityMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("id = ? AND community_id = ? AND status = ?", memberID, communityID, models.CommunityMemberStatusActive).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCommunityMemberNotFound
	}
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *communityRepository) UpdateMemberRole(
	ctx context.Context,
	memberID string,
	role string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.CommunityMember{}).
		Where("id = ? AND status = ?", memberID, models.CommunityMemberStatusActive).
		Update("role", role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCommunityMemberNotFound
	}
	return nil
}

func (r *communityRepository) RemoveMember(
	ctx context.Context,
	memberID string,
) error {
	result := r.db.WithContext(ctx).
		Model(&models.CommunityMember{}).
		Where("id = ? AND status = ?", memberID, models.CommunityMemberStatusActive).
		Update("status", models.CommunityMemberStatusRemoved)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrCommunityMemberNotFound
	}
	return nil
}

func (r *communityRepository) Follow(
	ctx context.Context,
	communityID string,
	userID string,
) error {
	var exists int64
	if err := r.db.WithContext(ctx).
		Model(&models.Community{}).
		Where("id = ? AND status = ?", communityID, models.CommunityStatusActive).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists != 1 {
		return gorm.ErrRecordNotFound
	}
	follow := &models.CommunityFollow{CommunityID: communityID, UserID: userID}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(follow).Error
}

func (r *communityRepository) Unfollow(
	ctx context.Context,
	communityID string,
	userID string,
) error {
	return r.db.WithContext(ctx).
		Where("community_id = ? AND user_id = ?", communityID, userID).
		Delete(&models.CommunityFollow{}).Error
}

func (r *communityRepository) IsFollowing(
	ctx context.Context,
	communityID string,
	userID string,
) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.CommunityFollow{}).
		Where("community_id = ? AND user_id = ?", communityID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *communityRepository) CountFollowers(
	ctx context.Context,
	communityID string,
) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.CommunityFollow{}).
		Where("community_id = ?", communityID).
		Count(&count).Error
	return count, err
}

func (r *communityRepository) ListFollowing(
	ctx context.Context,
	userID string,
) ([]models.Community, error) {
	var communities []models.Community
	err := r.db.WithContext(ctx).
		Model(&models.Community{}).
		Joins("JOIN community_follows ON community_follows.community_id = communities.id").
		Where(
			"community_follows.user_id = ? AND communities.status = ?",
			userID,
			models.CommunityStatusActive,
		).
		Order("community_follows.created_at DESC").
		Find(&communities).Error
	return communities, err
}

type communityRepository struct {
	db *gorm.DB
}

func NewCommunityRepository(db *gorm.DB) CommunityRepository {
	return &communityRepository{db: db}
}

func (r *communityRepository) CreateWithOwner(
	ctx context.Context,
	community *models.Community,
	owner *models.CommunityMember,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(community).Error; err != nil {
			return err
		}
		owner.CommunityID = community.ID
		return tx.Create(owner).Error
	})
}

func (r *communityRepository) FindByID(
	ctx context.Context,
	id string,
) (*models.Community, error) {
	var community models.Community
	if err := r.db.WithContext(ctx).First(&community, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &community, nil
}

func (r *communityRepository) FindBySlug(
	ctx context.Context,
	slug string,
) (*models.Community, error) {
	var community models.Community
	if err := r.db.WithContext(ctx).
		First(
			&community,
			"slug = ? AND status = ?",
			slug,
			models.CommunityStatusActive,
		).Error; err != nil {
		return nil, err
	}
	return &community, nil
}

func (r *communityRepository) ListPublic(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.Community, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&models.Community{}).
		Where("status = ?", models.CommunityStatusActive)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	var communities []models.Community
	err := query.
		Order("slug ASC").
		Limit(limit).
		Offset(max(offset, 0)).
		Find(&communities).Error
	return communities, total, err
}

func (r *communityRepository) ListForUser(
	ctx context.Context,
	userID string,
) ([]models.Community, error) {
	var communities []models.Community
	err := r.db.WithContext(ctx).
		Model(&models.Community{}).
		Joins(
			"JOIN community_members ON community_members.community_id = communities.id",
		).
		Where(
			"community_members.user_id = ? AND community_members.status = ?",
			userID,
			models.CommunityMemberStatusActive,
		).
		Order("communities.name ASC").
		Find(&communities).Error
	return communities, err
}

func (r *communityRepository) FindActiveMember(
	ctx context.Context,
	communityID string,
	userID string,
) (*models.CommunityMember, error) {
	var member models.CommunityMember
	err := r.db.WithContext(ctx).
		Where(
			"community_id = ? AND user_id = ? AND status = ?",
			communityID,
			userID,
			models.CommunityMemberStatusActive,
		).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCommunityMemberNotFound
	}
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *communityRepository) ListMembers(
	ctx context.Context,
	communityID string,
) ([]models.CommunityMember, error) {
	var members []models.CommunityMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("community_id = ? AND status = ?", communityID, models.CommunityMemberStatusActive).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *communityRepository) CreateInvitation(
	ctx context.Context,
	invitation *models.CommunityInvitation,
) error {
	return r.db.WithContext(ctx).Create(invitation).Error
}

func (r *communityRepository) FindInvitationByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*models.CommunityInvitation, error) {
	var invitation models.CommunityInvitation
	if err := r.db.WithContext(ctx).
		First(&invitation, "token_hash = ?", tokenHash).Error; err != nil {
		return nil, err
	}
	return &invitation, nil
}

func (r *communityRepository) AcceptInvitation(
	ctx context.Context,
	invitation *models.CommunityInvitation,
	userID string,
) (*models.CommunityMember, error) {
	member := &models.CommunityMember{
		CommunityID: invitation.CommunityID,
		UserID:      userID,
		Role:        invitation.Role,
		Status:      models.CommunityMemberStatusActive,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.CommunityMember
		err := tx.Where(
			"community_id = ? AND user_id = ? AND status = ?",
			invitation.CommunityID,
			userID,
			models.CommunityMemberStatusActive,
		).First(&existing).Error
		if err == nil {
			return ErrAlreadyCommunityMember
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		now := time.Now().UTC()
		result := tx.Model(&models.CommunityInvitation{}).
			Where(
				"id = ? AND accepted_at IS NULL AND expires_at > ?",
				invitation.ID,
				now,
			).
			Updates(map[string]interface{}{
				"accepted_at": now,
				"accepted_by": userID,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvitationUnavailable
		}

		return tx.Create(member).Error
	})
	if err != nil {
		return nil, err
	}
	return member, nil
}
