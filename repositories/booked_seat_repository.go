package repositories

import (
	"context"
	"errors"
	"fmt"
	"go-ticketing/models"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BookedSeatRepository struct {
	DB  *gorm.DB
	rdb *redis.Client
}

type PublicBookedSeat struct {
	SeatID  string `json:"seat_id"`
	EventID string `json:"event_id"`
}

func NewBookedSeatRepository(db *gorm.DB, rdb *redis.Client) *BookedSeatRepository {
	return &BookedSeatRepository{DB: db, rdb: rdb}
}

func (r *BookedSeatRepository) FindAll(showID string) ([]models.BookedSeat, error) {
	var bookedSeats []models.BookedSeat
	query := r.DB

	if showID != "" {
		query = query.Where("event_id = ?", showID)
	}

	err := query.Preload("Ticket").Preload("Seat").Find(&bookedSeats).Error
	return bookedSeats, err
}

func (r *BookedSeatRepository) FindPublicByEvent(showID string) ([]PublicBookedSeat, error) {
	var bookedSeats []PublicBookedSeat
	eventID := showID

	if showID != "" {
		var event models.Event
		if err := r.DB.First(&event, "id = ? OR slug = ?", showID, showID).Error; err == nil {
			eventID = event.ID
		}
	}

	query := r.DB.Model(&models.BookedSeat{}).Select("seat_id", "event_id")
	if eventID != "" {
		query = query.Where("event_id = ?", eventID)
	}

	err := query.Find(&bookedSeats).Error
	return bookedSeats, err
}

func (r *BookedSeatRepository) FindByTicket(eventID string, ticketID string) (*models.BookedSeat, error) {
	var event models.Event
	if err := r.DB.First(&event, "id = ? OR slug = ?", eventID, eventID).Error; err != nil {
		return nil, err
	}

	var bookedSeat models.BookedSeat
	err := r.DB.
		Preload("Seat").
		First(&bookedSeat, "event_id = ? AND ticket_id = ?", event.ID, ticketID).
		Error
	return &bookedSeat, err
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
	ctx := context.Background()

	for _, seat := range seats {
		var key = fmt.Sprintf("seat_lock:%s:%s", seat.EventID, seat.SeatID)

		if seat.ID != "" {
			var existing models.BookedSeat
			err := r.DB.First(&existing, "id = ?", seat.ID).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// Create baru
					if err := r.DB.Create(&seat).Error; err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				// Update existing
				if err := r.DB.Model(&existing).Updates(seat).Error; err != nil {
					return nil, err
				}
			}
		} else {
			// Create baru
			if err := r.DB.Create(&seat).Error; err != nil {
				return nil, err
			}
		}

		// Ambil kembali dengan preload relasi Ticket dan Seat
		var full models.BookedSeat
		if err := r.DB.
			Preload("Ticket").
			Preload("Seat").
			First(&full, "id = ?", seat.ID).
			Error; err != nil {
			return nil, err
		}

		result = append(result, full)

		// 🔓 Unlock seat in Redis
		if err := r.rdb.Del(ctx, key).Err(); err != nil {
			return nil, fmt.Errorf("failed to unlock seat %s: %w", key, err)
		}
	}

	return result, nil
}

func (r *BookedSeatRepository) ConfirmBooking(ctx context.Context, eventID string, seatID string, ticketID string) (*models.BookedSeat, error) {
	event, seat, ticket, err := r.validateConfirmBooking(eventID, seatID, ticketID)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("seat_lock:%s:%s", event.ID, seatID)
	owner, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil || owner != ticketID {
		return nil, errors.New("sesi pemilihan kursi telah habis atau kursi dipilih orang lain")
	}
	if err != nil {
		return nil, err
	}

	var booked *models.BookedSeat
	err = r.DB.Transaction(func(tx *gorm.DB) error {
		var ticketCount int64
		if err := tx.Model(&models.BookedSeat{}).
			Where("event_id = ? AND ticket_id = ?", event.ID, ticket.ID).
			Count(&ticketCount).Error; err != nil {
			return err
		}
		if ticketCount > 0 {
			return errors.New("tiket ini sudah digunakan untuk membooking kursi")
		}

		var seatCount int64
		if err := tx.Model(&models.BookedSeat{}).
			Where("event_id = ? AND seat_id = ?", event.ID, seat.ID).
			Count(&seatCount).Error; err != nil {
			return err
		}
		if seatCount > 0 {
			return errors.New("kursi sudah dibooking")
		}

		booked = &models.BookedSeat{
			EventID:  event.ID,
			SeatID:   seat.ID,
			TicketID: ticket.ID,
			Name:     ticket.Name,
			AdminID:  ticket.ID,
		}

		if err := tx.Create(booked).Error; err != nil {
			if isDuplicateError(err) {
				return errors.New("kursi atau tiket sudah dibooking")
			}
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var full models.BookedSeat
	if err := r.DB.Preload("Ticket").Preload("Seat").First(&full, "id = ?", booked.ID).Error; err != nil {
		return nil, err
	}

	// Remove seat lock and user lock
	userLockKey := fmt.Sprintf("user_lock:%s:%s", event.ID, ticketID)
	r.rdb.Del(ctx, key, userLockKey)

	return &full, nil
}

func (r *BookedSeatRepository) validateConfirmBooking(eventID string, seatID string, ticketID string) (*models.Event, *models.Seat, *models.Ticket, error) {
	if eventID == "" || seatID == "" || ticketID == "" {
		return nil, nil, nil, errors.New("event_id dan seat_id wajib diisi")
	}

	var event models.Event
	if err := r.DB.First(&event, "id = ? OR slug = ?", eventID, eventID).Error; err != nil {
		return nil, nil, nil, errors.New("event tidak ditemukan")
	}
	if !strings.EqualFold(event.Status, "active") {
		return nil, nil, nil, errors.New("event tidak aktif")
	}
	if event.WarStartDate != nil && time.Now().Before(*event.WarStartDate) {
		return nil, nil, nil, errors.New("war kursi belum dimulai")
	}

	var seat models.Seat
	if err := r.DB.First(&seat, "id = ? AND event_id = ?", seatID, event.ID).Error; err != nil {
		return nil, nil, nil, errors.New("kursi tidak ditemukan untuk event ini")
	}
	if strings.EqualFold(seat.Category, "STAGE") {
		return nil, nil, nil, errors.New("area ini tidak dapat dipilih")
	}

	var ticket models.Ticket
	if err := r.DB.First(&ticket, "id = ? AND event_id = ?", ticketID, event.ID).Error; err != nil {
		return nil, nil, nil, errors.New("tiket tidak valid untuk event ini")
	}
	if !seatGenderAllowed(ticket.Gender, seat.Gender) {
		return nil, nil, nil, errors.New("kursi ini tidak sesuai gender tiket")
	}
	if !seatCategoryAllowed(ticket.Category, seat.Category) {
		return nil, nil, nil, errors.New("kursi ini tidak sesuai kategori tiket")
	}

	return &event, &seat, &ticket, nil
}

func isDuplicateError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique index") ||
		strings.Contains(msg, "sqlstate 23505")
}
