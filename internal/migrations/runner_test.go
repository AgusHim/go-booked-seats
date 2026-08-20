package migrations

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunnerApplyIsIdempotentAndVerifiable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	var output bytes.Buffer
	runner, err := NewRunner(db, &output)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}

	ctx := context.Background()
	if err := runner.Apply(ctx, false); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if !strings.Contains(output.String(), "applied 000001_baseline") {
		t.Fatalf("expected migration output, got %q", output.String())
	}

	output.Reset()
	if err := runner.Apply(ctx, false); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("second apply should be a no-op, got %q", output.String())
	}

	if err := runner.Verify(ctx); err != nil {
		t.Fatalf("verify: %v", err)
	}

	var count int64
	if err := db.Table("schema_migrations").Count(&count).Error; err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	migrations, err := Load()
	if err != nil {
		t.Fatalf("load migrations: %v", err)
	}
	if count != int64(len(migrations)) {
		t.Fatalf("expected %d applied migrations, got %d", len(migrations), count)
	}
}

func TestDryRunDoesNotCreateMigrationTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:dry-run?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	var output bytes.Buffer
	runner, err := NewRunner(db, &output)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}

	if err := runner.Apply(context.Background(), true); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if db.Migrator().HasTable("schema_migrations") {
		t.Fatal("dry run must not create schema_migrations")
	}
	if !strings.Contains(output.String(), "would apply 000001_baseline") {
		t.Fatalf("expected dry-run output, got %q", output.String())
	}
}
