package backfill

import (
	"testing"
	"time"

	"go-ticketing/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackfillEventTenantsIsDryRunSafeAndIdempotent(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:event-tenant-backfill?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Community{},
		&models.CommunityMember{},
		&models.Event{},
		&models.EventCommunityAssignment{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := &models.User{
		Name:     "Legacy Owner",
		Email:    "owner@example.test",
		Password: "hash",
	}
	if err := db.Create(owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	otherCommunity := &models.Community{
		Name: "Other Community",
		Slug: "other-community",
		Type: models.CommunityTypeGeneral,
	}
	if err := db.Create(otherCommunity).Error; err != nil {
		t.Fatalf("create other community: %v", err)
	}

	events := []models.Event{
		{Name: "Legacy One", Slug: "legacy-one", Date: time.Now().UTC()},
		{Name: "Legacy Two", Slug: "legacy-two", Date: time.Now().UTC()},
		{Name: "Already Owned", Slug: "already-owned", Date: time.Now().UTC()},
	}
	for index := range events {
		if err := db.Create(&events[index]).Error; err != nil {
			t.Fatalf("create event: %v", err)
		}
	}
	existing := &models.EventCommunityAssignment{
		EventID:     events[2].ID,
		CommunityID: otherCommunity.ID,
		Source:      "manual",
		AssignedAt:  time.Now().UTC(),
	}
	if err := db.Create(existing).Error; err != nil {
		t.Fatalf("create existing assignment: %v", err)
	}

	options := EventTenantOptions{
		OwnerEmail:    owner.Email,
		CommunityName: "Legacy Usloop",
		CommunitySlug: "legacy-usloop",
		CommunityType: models.CommunityTypeDakwah,
	}
	dryRun, err := BackfillEventTenants(db, options)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !dryRun.WouldCreateTenant || dryRun.TotalEvents != 3 ||
		dryRun.AlreadyAssigned != 1 || dryRun.UnassignedEvents != 2 ||
		dryRun.AssignedEvents != 0 {
		t.Fatalf("unexpected dry-run report: %#v", dryRun)
	}
	var communityCount int64
	if err := db.Model(&models.Community{}).
		Where("slug = ?", options.CommunitySlug).
		Count(&communityCount).Error; err != nil {
		t.Fatalf("count communities: %v", err)
	}
	if communityCount != 0 {
		t.Fatal("dry run must not create a community")
	}

	options.Apply = true
	applied, err := BackfillEventTenants(db, options)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if applied.AssignedEvents != 2 || applied.CommunityID == "" {
		t.Fatalf("unexpected apply report: %#v", applied)
	}

	var preserved models.EventCommunityAssignment
	if err := db.First(&preserved, "event_id = ?", events[2].ID).Error; err != nil {
		t.Fatalf("load preserved assignment: %v", err)
	}
	if preserved.CommunityID != otherCommunity.ID {
		t.Fatal("backfill must not overwrite an existing tenant assignment")
	}
	var ownerMembership models.CommunityMember
	if err := db.Where(
		"community_id = ? AND user_id = ?",
		applied.CommunityID,
		owner.ID,
	).First(&ownerMembership).Error; err != nil {
		t.Fatalf("load owner membership: %v", err)
	}
	if ownerMembership.Role != models.CommunityRoleOwner {
		t.Fatalf("unexpected owner role %q", ownerMembership.Role)
	}

	again, err := BackfillEventTenants(db, options)
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if again.AssignedEvents != 0 || again.UnassignedEvents != 0 {
		t.Fatalf("second apply must be a no-op: %#v", again)
	}
}
