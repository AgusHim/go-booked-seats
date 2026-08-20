package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"go-ticketing/config"
	"go-ticketing/internal/migrations"

	"github.com/joho/godotenv"
)

func main() {
	command := flag.String("command", "status", "migration command: up, status, or verify")
	dryRun := flag.Bool("dry-run", false, "show pending migrations without applying them")
	flag.Parse()

	_ = godotenv.Load(".env")
	db := config.ConnectDatabase()

	runner, err := migrations.NewRunner(db, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	switch *command {
	case "up":
		if err := runner.Apply(ctx, *dryRun); err != nil {
			log.Fatal(err)
		}
	case "status":
		statuses, err := runner.Status(ctx)
		if err != nil {
			log.Fatal(err)
		}
		for _, status := range statuses {
			state := "pending"
			if status.Applied {
				state = "applied"
			}
			fmt.Printf("%06d %-32s %s\n", status.Migration.Version, status.Migration.Name, state)
		}
	case "verify":
		if err := runner.Verify(ctx); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command %q; use up, status, or verify", *command)
	}
}
