package repositories_test

import (
	"context"
	"go-ticketing/config"
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
	repo := repositories.NewSeatRepository(db, rdb)

	ctx := context.Background()
	showID := "test-event-1"
	seatID := "seat-A1"

	// Cleanup before test
	rdb.Del(ctx, "seat_lock:"+showID+":"+seatID)

	var successCount int32
	var failCount int32
	var wg sync.WaitGroup

	numConcurrentRequests := 1000

	// Fire 1000 requests simultaneously
	for i := 0; i < numConcurrentRequests; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			uid := "user-" + string(rune(userID))
			status, err := repo.LockSeat(ctx, showID, seatID, uid, "")
			
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

	// Cleanup after test
	rdb.Del(ctx, "seat_lock:"+showID+":"+seatID)
}
