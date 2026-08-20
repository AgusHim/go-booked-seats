package main

import (
	"flag"
	"fmt"
	"log"

	"go-ticketing/config"
	"go-ticketing/internal/backfill"
	"go-ticketing/models"
)

func main() {
	var options backfill.EventTenantOptions
	flag.StringVar(&options.OwnerEmail, "owner-email", "", "existing user email that will own the legacy tenant")
	flag.StringVar(&options.CommunityName, "community-name", "", "legacy tenant display name")
	flag.StringVar(&options.CommunitySlug, "community-slug", "", "legacy tenant slug")
	flag.StringVar(&options.CommunityType, "community-type", models.CommunityTypeGeneral, "general, dakwah, or running")
	flag.BoolVar(&options.Apply, "apply", false, "apply changes; omit for a read-only dry run")
	flag.Parse()

	report, err := backfill.BackfillEventTenants(config.ConnectDatabase(), options)
	if err != nil {
		log.Fatal(err)
	}
	mode := "dry-run"
	if options.Apply {
		mode = "applied"
	}
	fmt.Printf(
		"%s: community_id=%s create_tenant=%t total_events=%d already_assigned=%d unassigned=%d assigned=%d\n",
		mode,
		report.CommunityID,
		report.WouldCreateTenant,
		report.TotalEvents,
		report.AlreadyAssigned,
		report.UnassignedEvents,
		report.AssignedEvents,
	)
}
