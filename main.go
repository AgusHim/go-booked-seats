package main

import (
	"context"
	"go-ticketing/config"
	"go-ticketing/models"
	"go-ticketing/routes"
	"log"
	"os"

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

	db.AutoMigrate(&models.Seat{}, &models.BookedSeat{}, &models.User{}, &models.Ticket{}) // Ini akan buat file data.db otomatis

	routes.RegisterRoutes(app, db, rdb)

	log.Fatal(app.Listen(":3000"))
}
