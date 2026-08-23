package main

import (
	"context"
	"encoding/json"
	"go-ticketing/config"
	"go-ticketing/models"
	"go-ticketing/routes"
	"go-ticketing/utils"
	ws "go-ticketing/websocket"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	_ = godotenv.Load(".env")
	if _, err := utils.JWTSecret(); err != nil {
		log.Fatalf("Invalid JWT configuration: %v", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: 5 * 1024 * 1024,
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins(),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false,
	}))
	db := config.ConnectDatabase()

	redisUrl := os.Getenv("REDIS_URL")
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	if rdb == nil {
		log.Fatal("Failed to connect Redis")
	}

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Enable keyspace events for expired keys
	if err := rdb.ConfigSet(context.Background(), "notify-keyspace-events", "Ex").Err(); err != nil {
		log.Printf("Failed to configure Redis keyspace events: %v", err)
	}

	// Start a background worker to listen for expired locks
	go func() {
		pubsub := rdb.Subscribe(context.Background(), "__keyevent@0__:expired")
		defer pubsub.Close()
		ch := pubsub.Channel()

		for msg := range ch {
			if strings.HasPrefix(msg.Payload, "seat_lock:") {
				parts := strings.Split(msg.Payload, ":")
				if len(parts) >= 3 {
					// Broadcast that the seat is now available
					payload, _ := json.Marshal(map[string]string{
						"seat_id": parts[2],
					})
					wsMsg := models.Message{
						Type:     "seat_unlocked",
						SenderID: "system",
						Payload:  payload,
					}
					msgBytes, _ := json.Marshal(wsMsg)
					ws.GetManager().Broadcast(msgBytes)
				}
			}
		}
	}()

	if err := db.AutoMigrate(&models.Event{}, &models.Seat{}, &models.BookedSeat{}, &models.User{}, &models.Ticket{}, &models.Setting{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Drop the deprecated `event_scanner_user_full_name` column. Scanner name
	// is now derived from the logged-in account performing the claim, not from
	// a per-event field. Idempotent: only runs while the column still exists.
	if db.Migrator().HasColumn(&models.Event{}, "event_scanner_user_full_name") {
		if err := db.Migrator().DropColumn(&models.Event{}, "event_scanner_user_full_name"); err != nil {
			log.Printf("failed to drop event_scanner_user_full_name column: %v", err)
		}
	}

	routes.RegisterRoutes(app, db, rdb)

	log.Fatal(app.Listen(":3000"))
}

func allowedOrigins() string {
	origins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if origins != "" {
		return origins
	}
	return "http://127.0.0.1:5173,http://localhost:5173"
}
