package repositories

import (
	"context"
	"errors"
	"fmt"
	"go-ticketing/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BookedSeatRepository struct {
	DB  *gorm.DB
	rdb *redis.Client
}

func NewBookedSeatRepository(db *gorm.DB, rdb *redis.Client) *BookedSeatRepository {
	return &BookedSeatRepository{DB: db, rdb: rdb}
}

func (r *BookedSeatRepository) FindAll(showID string) ([]models.BookedSeat, error) {
	var bookedSeats []models.BookedSeat
	query := r.DB

	if showID != "" {
		query = query.Where("show_id = ?", showID)
	}

	err := query.Find(&bookedSeats).Error
	return bookedSeats, err
}

func (r *BookedSeatRepository) FindByID(id string) (*models.BookedSeat, error) {
	var bookedSeat models.BookedSeat
	err := r.DB.First(&bookedSeat, "id = ?", id).Error
	return &bookedSeat, err
}

func (r *BookedSeatRepository) Create(bookedSeat *models.BookedSeat) error {
	return r.DB.Create(bookedSeat).Error
}

func (r *BookedSeatRepository) Update(bookedSeat *models.BookedSeat) error {
	return r.DB.Omit("seat").Save(bookedSeat).Error
}

func (r *BookedSeatRepository) Delete(id string) error {
	return r.DB.Delete(&models.BookedSeat{}, "id = ?", id).Error
}

func (r *BookedSeatRepository) UpsertBookedSeats(seats []models.BookedSeat) ([]models.BookedSeat, error) {
	var result []models.BookedSeat

	for _, seat := range seats {
		var key = fmt.Sprintf("%s:%s", seat.ShowID, seat.SeatID)
		if seat.ID != "" {
			var existing models.BookedSeat
			err := r.DB.First(&existing, "id = ?", seat.ID).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					if err := r.DB.Create(&seat).Error; err != nil {
						return nil, err
					}
					result = append(result, seat)
				} else {
					return nil, err
				}
			} else {
				if err := r.DB.Model(&existing).Updates(seat).Error; err != nil {
					return nil, err
				}
				// reload updated record
				var updated models.BookedSeat
				if err := r.DB.First(&updated, "id = ?", seat.ID).Error; err != nil {
					return nil, err
				}
				result = append(result, updated)
			}
		} else {
			if err := r.DB.Create(&seat).Error; err != nil {
				return nil, err
			}
			result = append(result, seat)
		}
		ctx := context.Background()
		// 🔓 Unlock seat in Redis after upsert success
		if err := r.rdb.Del(ctx, key).Err(); err != nil {
			return nil, fmt.Errorf("failed to unlock seat %s: %w", key, err)
		}
	}

	return result, nil
}
