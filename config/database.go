package config

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func ConnectDatabase() *gorm.DB {
	postgresURL := os.Getenv("POSTGRE_URL")
	fmt.Printf("POSTGRE_URL %s", postgresURL)
	if postgresURL != "" {
		// Connect to PostgreSQL
		DB, err := gorm.Open(postgres.Open(postgresURL), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL: %v", err)
		}
		fmt.Println("Connected to PostgreSQL")
		return DB
	} else {
		// Fallback to SQLite
		DB, err := gorm.Open(sqlite.Open("data.db"), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to SQLite: %v", err)
		}
		fmt.Println("Connected to SQLite")
		return DB
	}
}
