package repositories

import (
	"go-ticketing/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindByTicketCodeNormalizesLookupAndChecksExternalID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.Ticket{}); err != nil {
		t.Fatalf("failed to migrate tickets: %v", err)
	}

	repo := NewTicketRepository(db)
	ticket := &models.Ticket{
		ID:          "ticket-1",
		TicketCode:  "0ZHVCXMT",
		ExtTicketID: "EXT-0ZHVCXMT",
		OrderID:     "#DTO8A8N4NYS",
		Name:        "Receipt Ticket",
	}
	if err := repo.Create(ticket); err != nil {
		t.Fatalf("failed to create ticket fixture: %v", err)
	}

	tests := []string{
		"0ZHVCXMT",
		" 0zhvcxmt ",
		"[0ZHVCXMT]",
		"ext-0zhvcxmt",
	}

	for _, tt := range tests {
		found, err := repo.FindByTicketCode(tt)
		if err != nil {
			t.Fatalf("expected lookup %q to find ticket: %v", tt, err)
		}
		if found.ID != ticket.ID {
			t.Fatalf("expected lookup %q to find ticket %s, got %s", tt, ticket.ID, found.ID)
		}
	}
}
