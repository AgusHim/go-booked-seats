package services

import (
	"testing"
	"time"

	"go-ticketing/models"
	"go-ticketing/repositories"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEventService_CreateAndUpdateScanner(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.Event{}); err != nil {
		t.Fatalf("failed to migrate events: %v", err)
	}

	repo := repositories.NewEventRepository(db)
	svc := NewEventService(repo)

	event := &models.Event{
		ID:             "event-test-1",
		Name:           "Workshop React",
		Location:       "Auditorium",
		Status:         "active",
		Date:           time.Now(),
		EventScannerID: "cmt12rzyl013js601r4p5kwj5",
	}

	if err := svc.CreateEvent(event); err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	fetched, err := svc.GetEvent(event.ID)
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}
	if fetched.EventScannerID != "cmt12rzyl013js601r4p5kwj5" {
		t.Fatalf("unexpected fetched scanner ID: %s", fetched.EventScannerID)
	}

	// Update scanner info
	updateInput := &models.Event{
		ID:             event.ID,
		Name:           "Workshop React Updated",
		Location:       "Auditorium 2",
		Status:         "active",
		EventScannerID: "scanner-new-id-999",
	}

	if err := svc.UpdateEvent(updateInput); err != nil {
		t.Fatalf("failed to update event: %v", err)
	}

	if updateInput.EventScannerID != "scanner-new-id-999" {
		t.Fatalf("updateInput was not updated: %+v", updateInput)
	}

	persisted, err := svc.GetEvent(event.ID)
	if err != nil {
		t.Fatalf("failed to get persisted event: %v", err)
	}
	if persisted.EventScannerID != "scanner-new-id-999" {
		t.Fatalf("unexpected persisted scanner ID: %s", persisted.EventScannerID)
	}
}
