package services

import (
	"context"
	"errors"
	"testing"

	"go-ticketing/models"
	"go-ticketing/repositories"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCommunityServiceInvitationRequiresMatchingEmail(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:community-service?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Community{},
		&models.CommunityMember{},
		&models.CommunityInvitation{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := &models.User{Name: "Owner", Email: "owner@example.test", Password: "hash"}
	invitee := &models.User{Name: "Invitee", Email: "invitee@example.test", Password: "hash"}
	other := &models.User{Name: "Other", Email: "other@example.test", Password: "hash"}
	for _, user := range []*models.User{owner, invitee, other} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	communityRepo := repositories.NewCommunityRepository(db)
	userRepo := repositories.NewUserRepository(db)
	service := NewCommunityService(communityRepo, userRepo)
	ctx := context.Background()

	community := &models.Community{Name: "Masjid Test", Type: models.CommunityTypeDakwah}
	if err := service.Create(ctx, community, owner.ID); err != nil {
		t.Fatalf("create community: %v", err)
	}

	result, err := service.Invite(
		ctx,
		community.ID,
		invitee.Email,
		models.CommunityRoleEventManager,
		owner.ID,
	)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if result.Token == "" || result.Invitation.TokenHash == result.Token {
		t.Fatal("invitation must return an opaque token and store only its hash")
	}

	if _, err := service.AcceptInvitation(ctx, result.Token, other.ID); !errors.Is(err, ErrInvitationEmail) {
		t.Fatalf("expected email mismatch, got %v", err)
	}

	member, err := service.AcceptInvitation(ctx, result.Token, invitee.ID)
	if err != nil {
		t.Fatalf("accept matching invitation: %v", err)
	}
	if member.CommunityID != community.ID || member.Role != models.CommunityRoleEventManager {
		t.Fatalf("unexpected membership: %#v", member)
	}
}

func TestCommunityServiceRejectsUnknownTemplateAndOwnerInvite(t *testing.T) {
	service := &communityService{}
	if err := service.Create(
		context.Background(),
		&models.Community{Name: "Invalid", Type: "unknown"},
		"user-id",
	); !errors.Is(err, ErrInvalidCommunityType) {
		t.Fatalf("expected invalid type, got %v", err)
	}

	service.communityRepo = &panicCommunityRepository{}
	if _, err := service.Invite(
		context.Background(),
		"community",
		"person@example.test",
		models.CommunityRoleOwner,
		"owner",
	); !errors.Is(err, ErrInvalidCommunityRole) {
		t.Fatalf("expected invalid owner invitation, got %v", err)
	}
}

func TestCommunityServiceProfileAndMemberManagementGuards(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:community-management?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Community{},
		&models.CommunityMember{},
		&models.CommunityInvitation{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	owner := &models.User{Name: "Owner", Email: "owner-manage@example.test", Password: "hash"}
	admin := &models.User{Name: "Admin", Email: "admin-manage@example.test", Password: "hash"}
	manager := &models.User{Name: "Manager", Email: "manager@example.test", Password: "hash"}
	for _, user := range []*models.User{owner, admin, manager} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	community := &models.Community{Name: "Before", Type: models.CommunityTypeDakwah}
	if err := db.Create(community).Error; err != nil {
		t.Fatalf("create community: %v", err)
	}
	members := []*models.CommunityMember{
		{CommunityID: community.ID, UserID: owner.ID, Role: models.CommunityRoleOwner},
		{CommunityID: community.ID, UserID: admin.ID, Role: models.CommunityRoleAdmin},
		{CommunityID: community.ID, UserID: manager.ID, Role: models.CommunityRoleEventManager},
	}
	for _, member := range members {
		if err := db.Create(member).Error; err != nil {
			t.Fatalf("create member: %v", err)
		}
	}
	service := NewCommunityService(
		repositories.NewCommunityRepository(db),
		repositories.NewUserRepository(db),
	)
	ctx := context.Background()

	updated, err := service.UpdateProfile(ctx, community.ID, CommunityProfileInput{
		Name:        "Majelis Updated",
		Description: "Deskripsi baru",
		Location:    "Solo",
		LogoURL:     "https://cdn.example.test/logo.png",
	})
	if err != nil || updated.Name != "Majelis Updated" {
		t.Fatalf("update profile=%#v err=%v", updated, err)
	}
	if _, err := service.UpdateProfile(ctx, community.ID, CommunityProfileInput{
		Name:    "Invalid URL",
		LogoURL: "javascript:alert(1)",
	}); !errors.Is(err, ErrInvalidCommunity) {
		t.Fatalf("expected invalid URL, got %v", err)
	}

	if err := service.UpdateMemberRole(
		ctx,
		community.ID,
		members[1].ID,
		models.CommunityRoleEventManager,
		admin.ID,
		models.CommunityRoleAdmin,
	); !errors.Is(err, ErrCannotManageSelf) {
		t.Fatalf("admin self-change should fail, got %v", err)
	}
	if err := service.UpdateMemberRole(
		ctx,
		community.ID,
		members[2].ID,
		models.CommunityRoleAdmin,
		admin.ID,
		models.CommunityRoleAdmin,
	); !errors.Is(err, ErrMemberRoleEscalation) {
		t.Fatalf("admin escalation should fail, got %v", err)
	}
	if err := service.UpdateMemberRole(
		ctx,
		community.ID,
		members[2].ID,
		models.CommunityRoleModerator,
		admin.ID,
		models.CommunityRoleAdmin,
	); err != nil {
		t.Fatalf("admin should manage lower role: %v", err)
	}
	if err := service.RemoveMember(
		ctx,
		community.ID,
		members[0].ID,
		admin.ID,
		models.CommunityRoleAdmin,
	); !errors.Is(err, ErrOwnerImmutable) {
		t.Fatalf("owner removal should fail, got %v", err)
	}
	if err := service.RemoveMember(
		ctx,
		community.ID,
		members[2].ID,
		owner.ID,
		models.CommunityRoleOwner,
	); err != nil {
		t.Fatalf("owner should remove non-owner: %v", err)
	}
}

type panicCommunityRepository struct {
	repositories.CommunityRepository
}

func (p *panicCommunityRepository) CreateInvitation(
	ctx context.Context,
	invitation *models.CommunityInvitation,
) error {
	panic("CreateInvitation should not be called for invalid role")
}
