//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Connect to SQLite database
	dbFilePath := "data.db"
	fmt.Printf("Connecting to SQLite (%s)...\n", dbFilePath)
	db, err := gorm.Open(sqlite.Open(dbFilePath), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Read SQL file
	sqlFilePath := "../k6-test/mock_war_kursi_postgres.sql"
	fmt.Printf("Reading SQL file from %s...\n", sqlFilePath)
	sqlBytes, err := os.ReadFile(sqlFilePath)
	if err != nil {
		log.Fatalf("Failed to read SQL file: %v", err)
	}

	// Execute SQL
	fmt.Println("Executing SQL script...")
	result := db.Exec(string(sqlBytes))
	if result.Error != nil {
		log.Fatalf("Error executing SQL script: %v", result.Error)
	}

	fmt.Println("Successfully uploaded mock data!")
}
