package repositories_test

import (
	"context"
	"fmt"
	"go-ticketing/config"
	"go-ticketing/models"
	"go-ticketing/repositories"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func TestSeatLockingConcurrency(t *testing.T) {
	// Setup Redis Connection
	_ = godotenv.Load("../.env")
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		redisUrl = "localhost:6379" // Fallback
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis is not available, skipping concurrency test")
	}

	// Setup DB connection (we just need the interface to initialize SeatRepository, although LockSeat only uses Redis)
	db := config.ConnectDatabase()
	if err := db.AutoMigrate(&models.Event{}, &models.Seat{}, &models.BookedSeat{}, &models.User{}, &models.Ticket{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	repo := repositories.NewSeatRepository(db, rdb)

	ctx := context.Background()
	showID := "test-event-1"
	seatID := "seat-A1"
	_ = db.Where("id = ?", seatID).Delete(&models.Seat{}).Error
	_ = db.Where("id = ?", showID).Delete(&models.Event{}).Error
	if err := db.Create(&models.Event{
		ID:       showID,
		Name:     "Test Event",
		Location: "Test Venue",
		Status:   "active",
	}).Error; err != nil {
		t.Fatalf("failed to create event fixture: %v", err)
	}
	if err := db.Create(&models.Seat{
		ID:       seatID,
		EventID:  showID,
		Position: "A-1",
		Color:    "#fff",
		Category: "VIP",
		Gender:   "both",
	}).Error; err != nil {
		t.Fatalf("failed to create seat fixture: %v", err)
	}

	// Cleanup before test
	rdb.Del(ctx, "seat_lock:"+showID+":"+seatID)
	defer rdb.Del(ctx, "seat_lock:"+showID+":"+seatID)
	defer db.Where("id = ?", seatID).Delete(&models.Seat{})
	defer db.Where("id = ?", showID).Delete(&models.Event{})

	var successCount int32
	var failCount int32
	var wg sync.WaitGroup

	numConcurrentRequests := 1000

	// Fire 1000 requests simultaneously
	for i := 0; i < numConcurrentRequests; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			uid := fmt.Sprintf("user-%d", userID)
			status, err := repo.LockSeat(ctx, showID, seatID, uid, "", true)

			if err == nil && status == "locked" {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("Expected exactly 1 success, got %d", successCount)
	}
	if failCount != int32(numConcurrentRequests-1) {
		t.Errorf("Expected exactly %d failures, got %d", numConcurrentRequests-1, failCount)
	}

}

func TestTicketLockIsIdempotentUntilExplicitUnlock(t *testing.T) {
	_ = godotenv.Load("../.env")
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		redisUrl = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not available, skipping idempotency test")
	}

	db := config.ConnectDatabase()
	if err := db.AutoMigrate(&models.Event{}, &models.Seat{}, &models.BookedSeat{}, &models.User{}, &models.Ticket{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	repo := repositories.NewSeatRepository(db, rdb)

	showID := "test-event-idempotent"
	seatID := "seat-IDEMPOTENT-A1"
	ticketID := "ticket-idempotent-1"
	seatKey := "seat_lock:" + showID + ":" + seatID
	userLockKey := "user_lock:" + showID + ":" + ticketID

	_ = rdb.Del(ctx, seatKey, userLockKey).Err()
	_ = db.Where("event_id = ?", showID).Delete(&models.BookedSeat{}).Error
	_ = db.Where("id = ?", ticketID).Delete(&models.Ticket{}).Error
	_ = db.Where("id = ?", seatID).Delete(&models.Seat{}).Error
	_ = db.Where("id = ?", showID).Delete(&models.Event{}).Error
	defer rdb.Del(ctx, seatKey, userLockKey)
	defer db.Where("event_id = ?", showID).Delete(&models.BookedSeat{})
	defer db.Where("id = ?", ticketID).Delete(&models.Ticket{})
	defer db.Where("id = ?", seatID).Delete(&models.Seat{})
	defer db.Where("id = ?", showID).Delete(&models.Event{})

	if err := db.Create(&models.Event{
		ID:       showID,
		Name:     "Test Event Idempotent",
		Location: "Test Venue",
		Status:   "active",
	}).Error; err != nil {
		t.Fatalf("failed to create event fixture: %v", err)
	}
	if err := db.Create(&models.Seat{
		ID:       seatID,
		EventID:  showID,
		Position: "A-1",
		Color:    "#fff",
		Category: "VIP",
		Gender:   "both",
	}).Error; err != nil {
		t.Fatalf("failed to create seat fixture: %v", err)
	}
	if err := db.Create(&models.Ticket{
		ID:         ticketID,
		TicketCode: "IDEMPOTENT-1",
		OrderID:    "IDEMPOTENT-ORDER-1",
		Name:       "Test Ticket",
		Category:   "VIP",
		Gender:     "male",
		EventID:    showID,
	}).Error; err != nil {
		t.Fatalf("failed to create ticket fixture: %v", err)
	}

	firstStatus, err := repo.LockSeat(ctx, showID, seatID, ticketID, "", false)
	if err != nil {
		t.Fatalf("first lock failed: %v", err)
	}
	if firstStatus != "locked" {
		t.Fatalf("expected first lock status locked, got %s", firstStatus)
	}

	secondStatus, err := repo.LockSeat(ctx, showID, seatID, ticketID, "", false)
	if err != nil {
		t.Fatalf("second lock failed: %v", err)
	}
	if secondStatus != "locked" {
		t.Fatalf("expected repeated lock status locked, got %s", secondStatus)
	}

	owner, err := rdb.Get(ctx, seatKey).Result()
	if err != nil {
		t.Fatalf("expected seat lock to remain after repeated lock: %v", err)
	}
	if owner != ticketID {
		t.Fatalf("expected seat lock owner %s, got %s", ticketID, owner)
	}

	unlockStatus, err := repo.LockSeat(ctx, showID, seatID, ticketID, "unlock", false)
	if err != nil {
		t.Fatalf("unlock failed: %v", err)
	}
	if unlockStatus != "unlocked" {
		t.Fatalf("expected unlock status unlocked, got %s", unlockStatus)
	}
	if _, err := rdb.Get(ctx, seatKey).Result(); err != redis.Nil {
		t.Fatalf("expected seat lock to be removed after explicit unlock, got err %v", err)
	}
}
