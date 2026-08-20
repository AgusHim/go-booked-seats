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

func TestPublicEventDiscoveryRequiresPublishedTenantAssignment(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:event-discovery?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Community{},
		&models.CommunityFollow{},
		&models.Event{},
		&models.EventCommunityAssignment{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := NewEventRepository(db)
	ctx := context.Background()

	user := &models.User{Name: "Follower", Email: "follower@example.test", Password: "hash"}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	dakwah := &models.Community{
		Name: "Majelis Solo",
		Slug: "majelis-solo",
		Type: models.CommunityTypeDakwah,
	}
	running := &models.Community{
		Name: "Solo Running",
		Slug: "solo-running",
		Type: models.CommunityTypeRunning,
	}
	inactive := &models.Community{
		Name:   "Inactive",
		Slug:   "inactive",
		Type:   models.CommunityTypeGeneral,
		Status: models.CommunityStatusInactive,
	}
	for _, community := range []*models.Community{dakwah, running, inactive} {
		if err := db.Create(community).Error; err != nil {
			t.Fatalf("create community: %v", err)
		}
	}

	now := time.Now().UTC()
	publishedDakwah := createDiscoveryEvent(t, db, "Kajian Akbar", "kajian-akbar", "published", now.Add(48*time.Hour))
	publishedRunning := createDiscoveryEvent(t, db, "Sunday Run", "sunday-run", "published", now.Add(24*time.Hour))
	draft := createDiscoveryEvent(t, db, "Draft Event", "draft-event", "draft", now.Add(12*time.Hour))
	unassigned := createDiscoveryEvent(t, db, "Unassigned Event", "unassigned-event", "published", now.Add(6*time.Hour))
	inactiveEvent := createDiscoveryEvent(t, db, "Inactive Event", "inactive-event", "published", now.Add(72*time.Hour))
	_ = unassigned

	assignDiscoveryEvent(t, db, publishedDakwah.ID, dakwah.ID)
	assignDiscoveryEvent(t, db, publishedRunning.ID, running.ID)
	assignDiscoveryEvent(t, db, draft.ID, dakwah.ID)
	assignDiscoveryEvent(t, db, inactiveEvent.ID, inactive.ID)
	if err := db.Create(&models.CommunityFollow{
		CommunityID: dakwah.ID,
		UserID:      user.ID,
	}).Error; err != nil {
		t.Fatalf("create follow: %v", err)
	}

	events, total, err := repo.ListPublished(ctx, PublicEventFilter{Query: "kajian", Limit: 10})
	if err != nil {
		t.Fatalf("list search: %v", err)
	}
	if total != 1 || len(events) != 1 || events[0].ID != publishedDakwah.ID {
		t.Fatalf("unexpected search result total=%d events=%#v", total, events)
	}
	if events[0].Community.ID != dakwah.ID || events[0].Community.Slug != dakwah.Slug {
		t.Fatalf("community summary missing: %#v", events[0].Community)
	}

	all, total, err := repo.ListPublished(ctx, PublicEventFilter{Limit: 1})
	if err != nil || total != 2 || len(all) != 1 || all[0].ID != publishedRunning.ID {
		t.Fatalf("pagination/order mismatch total=%d events=%#v err=%v", total, all, err)
	}
	if _, err := repo.FindPublished(ctx, draft.Slug); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("draft detail must not be public, got %v", err)
	}
	detail, err := repo.FindPublished(ctx, publishedDakwah.Slug)
	if err != nil || detail.ID != publishedDakwah.ID {
		t.Fatalf("published detail=%#v err=%v", detail, err)
	}

	personalized, err := repo.ListPublishedForFollowed(ctx, user.ID, 6)
	if err != nil || len(personalized) != 1 || personalized[0].ID != publishedDakwah.ID {
		t.Fatalf("personalized events=%#v err=%v", personalized, err)
	}
}

func createDiscoveryEvent(
	t *testing.T,
	db *gorm.DB,
	name string,
	slug string,
	status string,
	date time.Time,
) *models.Event {
	t.Helper()
	event := &models.Event{
		Name:     name,
		Slug:     slug,
		Status:   status,
		Date:     date,
		Location: "Solo",
	}
	if err := db.Create(event).Error; err != nil {
		t.Fatalf("create event: %v", err)
	}
	return event
}

func assignDiscoveryEvent(t *testing.T, db *gorm.DB, eventID, communityID string) {
	t.Helper()
	if err := db.Create(&models.EventCommunityAssignment{
		EventID:     eventID,
		CommunityID: communityID,
		Source:      "test",
		AssignedAt:  time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("assign event: %v", err)
	}
}
