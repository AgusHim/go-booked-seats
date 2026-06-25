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
	SaveBulkLayout(seats []models.Seat) error
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
		query = query.Where("event_id = ?", showID)
	}
	if err := query.Find(&seats).Error; err != nil {
		return nil, err
	}

	// Cache to Redis
	if data, err := json.Marshal(seats); err == nil {
		_ = r.rdb.Set(ctx, cacheKey, data, 24*time.Hour).Err()
	}

	return seats, nil
}

func (r *seatRepository) FindByID(id string) (models.Seat, error) {
	var seat models.Seat
	err := r.db.First(&seat, "id = ?", id).Error
	return seat, err
}

func (r *seatRepository) invalidateCache(eventID string) {
	ctx := context.Background()
	r.rdb.Del(ctx, fmt.Sprintf("seats:all:%s", eventID))
}

func (r *seatRepository) Create(seat models.Seat) error {
	err := r.db.Create(&seat).Error
	if err == nil {
		r.invalidateCache(seat.EventID)
	}
	return err
}

func (r *seatRepository) Update(seat models.Seat) error {
	err := r.db.Save(&seat).Error
	if err == nil {
		r.invalidateCache(seat.EventID)
	}
	return err
}

func (r *seatRepository) Delete(id string) error {
	var seat models.Seat
	r.db.First(&seat, "id = ?", id)
	err := r.db.Delete(&models.Seat{}, "id = ?", id).Error
	if err == nil && seat.EventID != "" {
		r.invalidateCache(seat.EventID)
	}
	return err
}

func (r *seatRepository) LockSeat(ctx context.Context, showID string, seatID string, userID string) (string, error) {
	key := fmt.Sprintf("seat_lock:%s:%s", showID, seatID)

	isAdmin := false
	var user models.User
	if err := r.db.Where("id = ?", userID).First(&user).Error; err == nil {
		if user.Role == "admin" {
			isAdmin = true
		}
	}

	userLockKey := fmt.Sprintf("user_lock:%s:%s", showID, userID)

	if !isAdmin {
		// Check if user already has a permanently booked seat
		var count int64
		r.db.Model(&models.BookedSeat{}).Where("ticket_id = ?", userID).Count(&count)
		if count > 0 {
			return "error", fmt.Errorf("anda sudah memiliki kursi yang dipesan secara permanen")
		}

		// Check if user already locked a different seat
		existingSeat, err := r.rdb.Get(ctx, userLockKey).Result()
		if err == nil && existingSeat != seatID {
			return "taken", fmt.Errorf("anda sudah mengunci kursi lain")
		}
	}

	currentOwner, err := r.rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		// Key belum ada → bisa lock (TTL 5 menit untuk war kursi)
		ok, err := r.rdb.SetNX(ctx, key, userID, 5*time.Minute).Result()
		if err != nil {
			return "error", err
		}
		if ok {
			if !isAdmin {
				r.rdb.Set(ctx, userLockKey, seatID, 5*time.Minute)
			}
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
		if !isAdmin {
			r.rdb.Del(ctx, userLockKey)
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
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, fmt.Sprintf("seat_lock:%s:*", showID), 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			val, err := r.rdb.Get(ctx, key).Result()
			if err == nil {
				// key is seat_lock:event_id:seat_id
				parts := strings.Split(key, ":")
				if len(parts) >= 3 {
					show := parts[1]
					seatID := parts[2]
					seat := &models.BookedSeat{
					ID:      key,
					EventID: show,
					SeatID:  seatID,
					AdminID: val,
				}
					locked = append(locked, seat)
				}
			}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return locked, nil
}

func (r *seatRepository) SaveBulkLayout(seats []models.Seat) error {
	for _, seat := range seats {
		if err := r.db.Save(&seat).Error; err != nil {
			return err
		}
	}
	if len(seats) > 0 {
		r.invalidateCache(seats[0].EventID)
	}
	return nil
}
