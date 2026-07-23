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

func TestMarkGoodieBagsClaimedOnlyReturnsNewlyClaimedForScanning(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.Ticket{}); err != nil {
		t.Fatalf("failed to migrate tickets: %v", err)
	}

	repo := NewTicketRepository(db)
	tickets := []*models.Ticket{
		{ID: "ticket-1", TicketCode: "TICKET-1", OrderID: "ORDER-1", Name: "Unclaimed"},
		{ID: "ticket-2", TicketCode: "TICKET-2", OrderID: "ORDER-2", Name: "Already Claimed", GoodieBagClaimed: true},
	}
	for _, ticket := range tickets {
		if err := repo.Create(ticket); err != nil {
			t.Fatalf("failed to create ticket fixture: %v", err)
		}
	}

	claimedTickets, newlyClaimed, err := repo.MarkGoodieBagsClaimed([]string{"ticket-1", "ticket-2"})
	if err != nil {
		t.Fatalf("failed to mark goodie bags claimed: %v", err)
	}
	if len(claimedTickets) != 2 {
		t.Fatalf("expected final response for 2 tickets, got %d", len(claimedTickets))
	}
	if len(newlyClaimed) != 1 || newlyClaimed[0].ID != "ticket-1" {
		t.Fatalf("expected only ticket-1 to be newly claimed, got %#v", newlyClaimed)
	}

	_, newlyClaimed, err = repo.MarkGoodieBagsClaimed([]string{"ticket-1", "ticket-2"})
	if err != nil {
		t.Fatalf("failed to repeat mark goodie bags claimed: %v", err)
	}
	if len(newlyClaimed) != 0 {
		t.Fatalf("expected repeated claim to be idempotent, got newly claimed %#v", newlyClaimed)
	}
}
