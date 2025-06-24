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
	FindAll() ([]models.Seat, error)
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

func (r *seatRepository) FindAll() ([]models.Seat, error) {
	var seats []models.Seat
	ctx := context.Background()
	cacheKey := "seats:all"

	// Check Redis
	cached, err := r.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		// Found in Redis
		if err := json.Unmarshal([]byte(cached), &seats); err == nil {
			return seats, nil
		}
	}

	// Not in Redis or failed unmarshal, fetch from DB
	err = r.db.Find(&seats).Error
	if err != nil {
		return nil, err
	}

	// Store to Redis
	data, err := json.Marshal(seats)
	if err == nil {
		_ = r.rdb.Set(ctx, cacheKey, data, time.Hour*2).Err() // Cache 5 menit
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
