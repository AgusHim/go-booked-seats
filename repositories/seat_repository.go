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
	LockSeat(ctx context.Context, showID string, seatID string, ownerID string, action string, isAdmin bool) (string, error)
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

func (r *seatRepository) LockSeat(ctx context.Context, showID string, seatID string, ownerID string, action string, isAdmin bool) (string, error) {
	if showID == "" || seatID == "" || ownerID == "" {
		return "error", fmt.Errorf("event_id, seat_id, and owner are required")
	}

	eventID, err := r.validateSeatLock(showID, seatID, ownerID, isAdmin)
	if err != nil {
		return "error", err
	}

	key := fmt.Sprintf("seat_lock:%s:%s", eventID, seatID)
	userLockKey := fmt.Sprintf("user_lock:%s:%s", eventID, ownerID)
	ttl := 5 * time.Minute

	if isAdmin {
		return r.lockSeatForAdmin(ctx, key, ownerID, action, ttl)
	}

	return r.lockSeatForTicket(ctx, key, userLockKey, seatID, ownerID, action, ttl)
}

func (r *seatRepository) validateSeatLock(eventID string, seatID string, ownerID string, isAdmin bool) (string, error) {
	var event models.Event
	if err := r.db.First(&event, "id = ? OR slug = ?", eventID, eventID).Error; err != nil {
		return "", fmt.Errorf("event tidak ditemukan")
	}
	if !strings.EqualFold(event.Status, "active") {
		return "", fmt.Errorf("event tidak aktif")
	}

	var seat models.Seat
	if err := r.db.First(&seat, "id = ? AND event_id = ?", seatID, event.ID).Error; err != nil {
		return "", fmt.Errorf("kursi tidak ditemukan untuk event ini")
	}
	if strings.EqualFold(seat.Category, "STAGE") {
		return "", fmt.Errorf("area ini tidak dapat dipilih")
	}

	var bookedCount int64
	if err := r.db.Model(&models.BookedSeat{}).
		Where("event_id = ? AND seat_id = ?", event.ID, seatID).
		Count(&bookedCount).Error; err != nil {
		return "", err
	}
	if bookedCount > 0 {
		return "", fmt.Errorf("kursi sudah dibooking")
	}

	if isAdmin {
		return event.ID, nil
	}

	if event.WarStartDate != nil && time.Now().Before(*event.WarStartDate) {
		return "", fmt.Errorf("war kursi belum dimulai")
	}

	var ticket models.Ticket
	if err := r.db.First(&ticket, "id = ? AND event_id = ?", ownerID, event.ID).Error; err != nil {
		return "", fmt.Errorf("tiket tidak valid untuk event ini")
	}

	var ticketBookedCount int64
	if err := r.db.Model(&models.BookedSeat{}).
		Where("event_id = ? AND ticket_id = ?", event.ID, ownerID).
		Count(&ticketBookedCount).Error; err != nil {
		return "", err
	}
	if ticketBookedCount > 0 {
		return "", fmt.Errorf("tiket ini sudah memiliki kursi")
	}

	if !seatGenderAllowed(ticket.Gender, seat.Gender) {
		return "", fmt.Errorf("kursi ini tidak sesuai gender tiket")
	}
	if !seatCategoryAllowed(ticket.Category, seat.Category) {
		return "", fmt.Errorf("kursi ini tidak sesuai kategori tiket")
	}

	return event.ID, nil
}

func (r *seatRepository) lockSeatForAdmin(ctx context.Context, key string, ownerID string, action string, ttl time.Duration) (string, error) {
	currentOwner, err := r.rdb.Get(ctx, key).Result()
	if action == "unlock" {
		if err == nil && currentOwner == ownerID {
			return "unlocked", r.rdb.Del(ctx, key).Err()
		}
		return "unlocked", nil
	}
	if err == redis.Nil {
		ok, err := r.rdb.SetNX(ctx, key, ownerID, ttl).Result()
		if err != nil {
			return "error", err
		}
		if ok {
			return "locked", nil
		}
		return "taken", nil
	}
	if err != nil {
		return "error", err
	}
	if currentOwner == ownerID {
		return "locked", nil
	}
	return "taken", nil
}

func (r *seatRepository) lockSeatForTicket(ctx context.Context, seatKey string, userLockKey string, seatID string, ownerID string, action string, ttl time.Duration) (string, error) {
	const script = `
local seatKey = KEYS[1]
local userLockKey = KEYS[2]
local seatID = ARGV[1]
local ownerID = ARGV[2]
local action = ARGV[3]
local ttlMs = tonumber(ARGV[4])

local currentOwner = redis.call("GET", seatKey)
if action == "unlock" then
	if currentOwner == ownerID then
		redis.call("DEL", seatKey)
		redis.call("DEL", userLockKey)
	end
	return "unlocked"
end

if currentOwner then
	if currentOwner == ownerID then
		redis.call("PEXPIRE", seatKey, ttlMs)
		redis.call("SET", userLockKey, seatID, "PX", ttlMs)
		return "locked"
	end
	return "taken"
end

local existingSeat = redis.call("GET", userLockKey)
if existingSeat and existingSeat ~= seatID then
	return "taken"
end

local ok = redis.call("SET", seatKey, ownerID, "PX", ttlMs, "NX")
if ok then
	redis.call("SET", userLockKey, seatID, "PX", ttlMs)
	return "locked"
end
return "taken"
`
	status, err := r.rdb.Eval(ctx, script, []string{seatKey, userLockKey}, seatID, ownerID, action, ttl.Milliseconds()).Text()
	if err != nil {
		return "error", err
	}
	return status, nil
}

func seatCategoryAllowed(ticketCategory string, seatCategory string) bool {
	if strings.EqualFold(seatCategory, "STAGE") {
		return false
	}
	if strings.TrimSpace(ticketCategory) == "" || strings.TrimSpace(seatCategory) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(ticketCategory), strings.TrimSpace(seatCategory))
}

func seatGenderAllowed(ticketGender string, seatGender string) bool {
	seat := normalizeGender(seatGender)
	if seat == "" || seat == "both" {
		return true
	}
	return normalizeGender(ticketGender) == seat
}

func normalizeGender(gender string) string {
	value := strings.ToLower(strings.TrimSpace(gender))
	switch value {
	case "", "both", "all", "semua":
		return "both"
	case "male", "m", "l", "laki-laki", "laki laki", "pria":
		return "male"
	case "female", "f", "p", "perempuan", "wanita":
		return "female"
	default:
		return value
	}
}

func (r *seatRepository) GetLockedSeats(ctx context.Context, showID string) ([]*models.BookedSeat, error) {
	scanEventID := showID
	var event models.Event
	if showID != "" && r.db.First(&event, "id = ? OR slug = ?", showID, showID).Error == nil {
		scanEventID = event.ID
	}

	cursor := uint64(0)
	var locked []*models.BookedSeat

	for {
		keys, nextCursor, err := r.rdb.Scan(ctx, cursor, fmt.Sprintf("seat_lock:%s:*", scanEventID), 100).Result()
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
