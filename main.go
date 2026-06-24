package main

import (
	"context"
	"encoding/json"
	"go-ticketing/config"
	"go-ticketing/models"
	"go-ticketing/routes"
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
	app := fiber.New()
	app.Use(cors.New())
	db := config.ConnectDatabase()

	redisUrl := os.Getenv("REDIS_URL")
	rdb := redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})

	if rdb == nil {
		log.Fatal("Failed to connect Redis")
	}

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Enable keyspace events for expired keys
	rdb.ConfigSet(context.Background(), "notify-keyspace-events", "Ex")

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

	db.AutoMigrate(&models.Event{}, &models.Seat{}, &models.BookedSeat{}, &models.User{}, &models.Ticket{}) // Ini akan buat file data.db otomatis

	routes.RegisterRoutes(app, db, rdb)

	log.Fatal(app.Listen(":3000"))
}
