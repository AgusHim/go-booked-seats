package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-ticketing/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCommunityRepositoryCreateListAndAcceptInvitation(t *testing.T) {
	db := openCommunityTestDB(t, "repository")
	repo := NewCommunityRepository(db)
	ctx := context.Background()

	owner := &models.User{Name: "Owner", Email: "owner@example.test", Password: "hash"}
	invitee := &models.User{Name: "Invitee", Email: "invitee@example.test", Password: "hash"}
	if err := db.Create(owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	if err := db.Create(invitee).Error; err != nil {
		t.Fatalf("create invitee: %v", err)
	}

	community := &models.Community{Name: "Komunitas Test", Type: models.CommunityTypeDakwah}
	ownerMember := &models.CommunityMember{
		UserID: owner.ID,
		Role:   models.CommunityRoleOwner,
		Status: models.CommunityMemberStatusActive,
	}
	if err := repo.CreateWithOwner(ctx, community, ownerMember); err != nil {
		t.Fatalf("create community: %v", err)
	}
	if community.Slug != "komunitas-test" {
		t.Fatalf("unexpected slug %q", community.Slug)
	}

	communities, err := repo.ListForUser(ctx, owner.ID)
	if err != nil {
		t.Fatalf("list communities: %v", err)
	}
	if len(communities) != 1 || communities[0].ID != community.ID {
		t.Fatalf("unexpected communities: %#v", communities)
	}

	invitation := &models.CommunityInvitation{
		CommunityID: community.ID,
		Email:       invitee.Email,
		Role:        models.CommunityRoleCheckinStaff,
		TokenHash:   "test-token-hash",
		InvitedBy:   owner.ID,
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	}
	if err := repo.CreateInvitation(ctx, invitation); err != nil {
		t.Fatalf("create invitation: %v", err)
	}

	member, err := repo.AcceptInvitation(ctx, invitation, invitee.ID)
	if err != nil {
		t.Fatalf("accept invitation: %v", err)
	}
	if member.Role != models.CommunityRoleCheckinStaff {
		t.Fatalf("unexpected member role %q", member.Role)
	}

	_, err = repo.AcceptInvitation(ctx, invitation, invitee.ID)
	if !errors.Is(err, ErrAlreadyCommunityMember) {
		t.Fatalf("expected already-member error, got %v", err)
	}
}

func TestCommunityFollowIsIdempotentAndScopedToUser(t *testing.T) {
	db := openCommunityTestDB(t, "follow")
	repo := NewCommunityRepository(db)
	ctx := context.Background()

	first := &models.User{Name: "First", Email: "first@example.test", Password: "hash"}
	second := &models.User{Name: "Second", Email: "second@example.test", Password: "hash"}
	for _, user := range []*models.User{first, second} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	community := &models.Community{Name: "Follow Test", Type: models.CommunityTypeDakwah}
	if err := db.Create(community).Error; err != nil {
		t.Fatalf("create community: %v", err)
	}

	if err := repo.Follow(ctx, community.ID, first.ID); err != nil {
		t.Fatalf("first follow: %v", err)
	}
	if err := repo.Follow(ctx, community.ID, first.ID); err != nil {
		t.Fatalf("idempotent follow: %v", err)
	}
	if err := repo.Follow(ctx, community.ID, second.ID); err != nil {
		t.Fatalf("second user follow: %v", err)
	}
	count, err := repo.CountFollowers(ctx, community.ID)
	if err != nil || count != 2 {
		t.Fatalf("follower count=%d err=%v", count, err)
	}

	if err := repo.Unfollow(ctx, community.ID, first.ID); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	firstFollowing, err := repo.IsFollowing(ctx, community.ID, first.ID)
	if err != nil || firstFollowing {
		t.Fatalf("first user should not follow, following=%v err=%v", firstFollowing, err)
	}
	secondFollowing, err := repo.IsFollowing(ctx, community.ID, second.ID)
	if err != nil || !secondFollowing {
		t.Fatalf("second user's follow must remain, following=%v err=%v", secondFollowing, err)
	}
	following, err := repo.ListFollowing(ctx, second.ID)
	if err != nil || len(following) != 1 || following[0].ID != community.ID {
		t.Fatalf("unexpected following list=%#v err=%v", following, err)
	}
}

func TestCommunityRepositoryListPublicOnlyReturnsActivePage(t *testing.T) {
	db := openCommunityTestDB(t, "public-list")
	repo := NewCommunityRepository(db)
	ctx := context.Background()
	for _, community := range []*models.Community{
		{Name: "Zulu Active", Type: models.CommunityTypeDakwah},
		{Name: "Alpha Active", Type: models.CommunityTypeRunning},
		{Name: "Hidden", Type: models.CommunityTypeGeneral, Status: models.CommunityStatusInactive},
	} {
		if err := db.Create(community).Error; err != nil {
			t.Fatalf("create community: %v", err)
		}
	}

	firstPage, total, err := repo.ListPublic(ctx, 1, 0)
	if err != nil {
		t.Fatalf("list first public page: %v", err)
	}
	if total != 2 || len(firstPage) != 1 || firstPage[0].Slug != "alpha-active" {
		t.Fatalf("unexpected first page=%#v total=%d", firstPage, total)
	}
	secondPage, total, err := repo.ListPublic(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list second public page: %v", err)
	}
	if total != 2 || len(secondPage) != 1 || secondPage[0].Slug != "zulu-active" {
		t.Fatalf("unexpected second page=%#v total=%d", secondPage, total)
	}
}

func openCommunityTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open("file:"+name+"?mode=memory&cache=shared"),
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
		&models.CommunityFollow{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}
