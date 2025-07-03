package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ticketing/models"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SeatRepository interface {
	FindAll(showID string) ([]models.Seat, error)
	FindByID(id string) (models.Seat, error)
	Create(seat models.Seat) error
	Update(seat models.Seat) error
	Delete(id string) error
	LockSeat(ctx context.Context, showID string, seatID string, userID string) (string, error)
	GetLockedSeats(ctx context.Context, showID string) ([]*models.BookedSeat, error)
}

type seatRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewSeatRepository(db *gorm.DB, rdb *redis.Client) SeatRepository {
	return &seatRepository{db: db, rdb: rdb}
}

func (r *seatRepository) FindAll(showID string) ([]models.Seat, error) {
	var seats []models.Seat
	ctx := context.Background()
	cacheKey := fmt.Sprintf("seats:all:%s", showID)

	// Attempt to fetch from Redis
	if cached, err := r.rdb.Get(ctx, cacheKey).Result(); err == nil {
		if err := json.Unmarshal([]byte(cached), &seats); err == nil {
			return seats, nil
		}
	}

	// Fallback to DB
	query := r.db
	if showID != "" {
		query = query.Where("show_id = ?", showID)
	}
	if err := query.Find(&seats).Error; err != nil {
		return nil, err
	}

	// Cache to Redis
	if data, err := json.Marshal(seats); err == nil {
		_ = r.rdb.Set(ctx, cacheKey, data, 5*time.Minute).Err()
	}

	return seats, nil
}

func (r *seatRepository) FindByID(id string) (models.Seat, error) {
	var seat models.Seat
	err := r.db.First(&seat, "id = ?", id).Error
	return seat, err
}

func (r *seatRepository) Create(seat models.Seat) error {
	return r.db.Create(&seat).Error
}

func (r *seatRepository) Update(seat models.Seat) error {
	return r.db.Save(&seat).Error
}

func (r *seatRepository) Delete(id string) error {
	return r.db.Delete(&models.Seat{}, "id = ?", id).Error
}

func (r *seatRepository) LockSeat(ctx context.Context, showID string, seatID string, userID string) (string, error) {
	key := fmt.Sprintf("%s:%s", showID, seatID)
	currentOwner, err := r.rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		// Key belum ada → bisa lock
		ok, err := r.rdb.SetNX(ctx, key, userID, 0).Result()
		if err != nil {
			return "error", err
		}
		if ok {
			return "locked", nil // sukses lock
		}
		return "error", nil // gagal lock tanpa sebab
	} else if err != nil {
		return "error", err
	}

	// Kursi sudah di-lock
	if currentOwner == userID {
		// User yang sama → lepaskan lock
		err := r.rdb.Del(ctx, key).Err()
		if err != nil {
			return "error", err
		}
		return "unlocked", nil
	}

	// Kursi di-lock user lain
	return "taken", nil
}

func (r *seatRepository) GetLockedSeats(ctx context.Context, showID string) ([]*models.BookedSeat, error) {
	cursor := uint64(0)
	var locked []*models.BookedSeat

	for {
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, fmt.Sprintf("%s:*", showID), 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			val, err := r.rdb.Get(ctx, key).Result()
			if err == nil {
				parts := strings.Split(key, ":")
				show := parts[0]
				seatID := parts[len(parts)-1]
				seat := &models.BookedSeat{
					ID:      key,
					ShowID:  show,
					SeatID:  seatID,
					AdminID: val,
				}

				locked = append(locked, seat)
			}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return locked, nil
}
