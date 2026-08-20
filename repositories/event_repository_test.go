package repositories

import (
	"go-ticketing/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEventScannerSettingsArePersisted(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.Event{}); err != nil {
		t.Fatalf("failed to migrate events: %v", err)
	}

	repo := NewEventRepository(db)
	event := &models.Event{
		ID:                       "event-1",
		Name:                     "Scanner Event",
		EventScannerID:           "scanner-123",
		EventScannerUserFullName: "Rijal",
	}
	if err := repo.CreateEvent(event); err != nil {
		t.Fatalf("failed to create event: %v", err)
	}

	event.EventScannerID = "scanner-456"
	event.EventScannerUserFullName = "Nadia"
	if err := repo.UpdateEvent(event); err != nil {
		t.Fatalf("failed to update event: %v", err)
	}

	persisted, err := repo.GetEvent(event.ID)
	if err != nil {
		t.Fatalf("failed to retrieve event: %v", err)
	}
	if persisted.EventScannerID != "scanner-456" {
		t.Fatalf("expected scanner ID to persist, got %q", persisted.EventScannerID)
	}
	if persisted.EventScannerUserFullName != "Nadia" {
		t.Fatalf("expected scanner user full name to persist, got %q", persisted.EventScannerUserFullName)
	}
}
