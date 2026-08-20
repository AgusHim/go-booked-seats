package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"go-ticketing/models"
	"go-ticketing/repositories"
)

var (
	ErrInvalidCommunity     = errors.New("invalid community")
	ErrInvalidCommunityType = errors.New("invalid community type")
	ErrInvalidCommunityRole = errors.New("invalid community role")
	ErrInvitationEmail      = errors.New("invitation belongs to a different email")
	ErrOwnerImmutable       = errors.New("community owner membership cannot be changed")
	ErrMemberRoleEscalation = errors.New("member role escalation is not allowed")
	ErrCannotManageSelf     = errors.New("member cannot change their own team access")
)

type InvitationResult struct {
	Invitation *models.CommunityInvitation `json:"invitation"`
	Token      string                      `json:"token"`
}

type CommunityPublicProfile struct {
	models.Community
	FollowerCount int64 `json:"follower_count"`
}

type CommunityProfileInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	CoverURL    string `json:"cover_url"`
	Location    string `json:"location"`
}

type CommunityService interface {
	Create(
		ctx context.Context,
		community *models.Community,
		ownerUserID string,
	) error
	GetPublic(ctx context.Context, slug string) (*CommunityPublicProfile, error)
	ListPublic(ctx context.Context, limit, offset int) ([]models.Community, int64, error)
	ListForUser(ctx context.Context, userID string) ([]models.Community, error)
	ListMembers(ctx context.Context, communityID string) ([]models.CommunityMember, error)
	Invite(
		ctx context.Context,
		communityID string,
		email string,
		role string,
		invitedBy string,
	) (*InvitationResult, error)
	AcceptInvitation(
		ctx context.Context,
		token string,
		userID string,
	) (*models.CommunityMember, error)
	Follow(ctx context.Context, communityID, userID string) error
	Unfollow(ctx context.Context, communityID, userID string) error
	IsFollowing(ctx context.Context, communityID, userID string) (bool, error)
	ListFollowing(ctx context.Context, userID string) ([]models.Community, error)
	GetPortal(ctx context.Context, communityID string) (*models.Community, error)
	UpdateProfile(
		ctx context.Context,
		communityID string,
		input CommunityProfileInput,
	) (*models.Community, error)
	UpdateMemberRole(
		ctx context.Context,
		communityID, memberID, role, actorID, actorRole string,
	) error
	RemoveMember(
		ctx context.Context,
		communityID, memberID, actorID, actorRole string,
	) error
}

func (s *communityService) GetPortal(
	ctx context.Context,
	communityID string,
) (*models.Community, error) {
	return s.communityRepo.FindByID(ctx, communityID)
}

func (s *communityService) ListPublic(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.Community, int64, error) {
	return s.communityRepo.ListPublic(ctx, limit, offset)
}

func (s *communityService) UpdateProfile(
	ctx context.Context,
	communityID string,
	input CommunityProfileInput,
) (*models.Community, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Location = strings.TrimSpace(input.Location)
	input.LogoURL = strings.TrimSpace(input.LogoURL)
	input.CoverURL = strings.TrimSpace(input.CoverURL)
	if input.Name == "" || len(input.Name) > 160 {
		return nil, fmt.Errorf("%w: name is required and must not exceed 160 characters", ErrInvalidCommunity)
	}
	if len(input.Description) > 5000 || len(input.Location) > 255 {
		return nil, fmt.Errorf("%w: profile field is too long", ErrInvalidCommunity)
	}
	for _, value := range []string{input.LogoURL, input.CoverURL} {
		if err := validateOptionalHTTPURL(value); err != nil {
			return nil, err
		}
	}
	community, err := s.communityRepo.FindByID(ctx, communityID)
	if err != nil {
		return nil, err
	}
	community.Name = input.Name
	community.Description = input.Description
	community.Location = input.Location
	community.LogoURL = input.LogoURL
	community.CoverURL = input.CoverURL
	if err := s.communityRepo.UpdateProfile(ctx, community); err != nil {
		return nil, err
	}
	return community, nil
}

func (s *communityService) UpdateMemberRole(
	ctx context.Context,
	communityID string,
	memberID string,
	role string,
	actorID string,
	actorRole string,
) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == models.CommunityRoleOwner || !models.IsCommunityRole(role) {
		return ErrInvalidCommunityRole
	}
	target, err := s.communityRepo.FindMemberByID(ctx, communityID, memberID)
	if err != nil {
		return err
	}
	if target.UserID == actorID {
		return ErrCannotManageSelf
	}
	if target.Role == models.CommunityRoleOwner {
		return ErrOwnerImmutable
	}
	if !canManageAdminRole(actorRole) &&
		(target.Role == models.CommunityRoleAdmin || role == models.CommunityRoleAdmin) {
		return ErrMemberRoleEscalation
	}
	return s.communityRepo.UpdateMemberRole(ctx, memberID, role)
}

func (s *communityService) RemoveMember(
	ctx context.Context,
	communityID string,
	memberID string,
	actorID string,
	actorRole string,
) error {
	target, err := s.communityRepo.FindMemberByID(ctx, communityID, memberID)
	if err != nil {
		return err
	}
	if target.UserID == actorID {
		return ErrCannotManageSelf
	}
	if target.Role == models.CommunityRoleOwner {
		return ErrOwnerImmutable
	}
	if !canManageAdminRole(actorRole) && target.Role == models.CommunityRoleAdmin {
		return ErrMemberRoleEscalation
	}
	return s.communityRepo.RemoveMember(ctx, memberID)
}

func canManageAdminRole(actorRole string) bool {
	return actorRole == models.CommunityRoleOwner || actorRole == "platform_admin"
}

type communityService struct {
	communityRepo repositories.CommunityRepository
	userRepo      repositories.UserRepository
	now           func() time.Time
}

func NewCommunityService(
	communityRepo repositories.CommunityRepository,
	userRepo repositories.UserRepository,
) CommunityService {
	return &communityService{
		communityRepo: communityRepo,
		userRepo:      userRepo,
		now:           time.Now,
	}
}

func (s *communityService) Create(
	ctx context.Context,
	community *models.Community,
	ownerUserID string,
) error {
	community.Name = strings.TrimSpace(community.Name)
	community.Type = strings.ToLower(strings.TrimSpace(community.Type))
	if community.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidCommunity)
	}
	if community.Type == "" {
		community.Type = models.CommunityTypeGeneral
	}
	if !models.IsCommunityType(community.Type) {
		return ErrInvalidCommunityType
	}
	community.Status = models.CommunityStatusActive

	owner := &models.CommunityMember{
		UserID: ownerUserID,
		Role:   models.CommunityRoleOwner,
		Status: models.CommunityMemberStatusActive,
	}
	return s.communityRepo.CreateWithOwner(ctx, community, owner)
}

func (s *communityService) GetPublic(
	ctx context.Context,
	slug string,
) (*CommunityPublicProfile, error) {
	community, err := s.communityRepo.FindBySlug(
		ctx,
		strings.ToLower(strings.TrimSpace(slug)),
	)
	if err != nil {
		return nil, err
	}
	count, err := s.communityRepo.CountFollowers(ctx, community.ID)
	if err != nil {
		return nil, err
	}
	return &CommunityPublicProfile{
		Community:     *community,
		FollowerCount: count,
	}, nil
}

func (s *communityService) ListForUser(
	ctx context.Context,
	userID string,
) ([]models.Community, error) {
	return s.communityRepo.ListForUser(ctx, userID)
}

func (s *communityService) ListMembers(
	ctx context.Context,
	communityID string,
) ([]models.CommunityMember, error) {
	return s.communityRepo.ListMembers(ctx, communityID)
}

func (s *communityService) Follow(
	ctx context.Context,
	communityID string,
	userID string,
) error {
	return s.communityRepo.Follow(ctx, communityID, userID)
}

func (s *communityService) Unfollow(
	ctx context.Context,
	communityID string,
	userID string,
) error {
	return s.communityRepo.Unfollow(ctx, communityID, userID)
}

func (s *communityService) IsFollowing(
	ctx context.Context,
	communityID string,
	userID string,
) (bool, error) {
	return s.communityRepo.IsFollowing(ctx, communityID, userID)
}

func (s *communityService) ListFollowing(
	ctx context.Context,
	userID string,
) ([]models.Community, error) {
	return s.communityRepo.ListFollowing(ctx, userID)
}

func (s *communityService) Invite(
	ctx context.Context,
	communityID string,
	email string,
	role string,
	invitedBy string,
) (*InvitationResult, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role == models.CommunityRoleOwner || !models.IsCommunityRole(role) {
		return nil, ErrInvalidCommunityRole
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate invitation token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := hashInvitationToken(token)

	invitation := &models.CommunityInvitation{
		CommunityID: communityID,
		Email:       normalizedEmail,
		Role:        role,
		TokenHash:   tokenHash,
		InvitedBy:   invitedBy,
		ExpiresAt:   s.now().UTC().Add(7 * 24 * time.Hour),
	}
	if err := s.communityRepo.CreateInvitation(ctx, invitation); err != nil {
		return nil, err
	}

	return &InvitationResult{Invitation: invitation, Token: token}, nil
}

func (s *communityService) AcceptInvitation(
	ctx context.Context,
	token string,
	userID string,
) (*models.CommunityMember, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, repositories.ErrInvitationUnavailable
	}

	invitation, err := s.communityRepo.FindInvitationByTokenHash(
		ctx,
		hashInvitationToken(token),
	)
	if err != nil {
		return nil, err
	}
	if invitation.AcceptedAt != nil || !invitation.ExpiresAt.After(s.now().UTC()) {
		return nil, repositories.ErrInvitationUnavailable
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(user.Email), invitation.Email) {
		return nil, ErrInvitationEmail
	}

	return s.communityRepo.AcceptInvitation(ctx, invitation, userID)
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || !strings.EqualFold(address.Address, value) {
		return "", errors.New("valid email is required")
	}
	return value, nil
}

func hashInvitationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func validateOptionalHTTPURL(value string) error {
	if value == "" {
		return nil
	}
	if len(value) > 2048 {
		return fmt.Errorf("%w: URL is too long", ErrInvalidCommunity)
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%w: logo and cover must use a valid HTTP(S) URL", ErrInvalidCommunity)
	}
	return nil
}
