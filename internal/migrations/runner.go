package migrations

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed sql/*.up.sql
var migrationFiles embed.FS

var migrationNamePattern = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.up\.sql$`)

type Migration struct {
	Version  int64
	Name     string
	Filename string
	SQL      string
	Checksum string
}

type AppliedMigration struct {
	Version   int64     `gorm:"column:version"`
	Name      string    `gorm:"column:name"`
	Checksum  string    `gorm:"column:checksum"`
	AppliedAt time.Time `gorm:"column:applied_at"`
}

type Status struct {
	Migration Migration
	Applied   bool
	AppliedAt *time.Time
}

type Runner struct {
	db  *gorm.DB
	out io.Writer
}

func NewRunner(db *gorm.DB, out io.Writer) (*Runner, error) {
	if db == nil {
		return nil, errors.New("migration runner requires a database")
	}
	if out == nil {
		return nil, errors.New("migration runner requires an output writer")
	}
	return &Runner{db: db, out: out}, nil
}

func Load() ([]Migration, error) {
	paths, err := fs.Glob(migrationFiles, "sql/*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("list embedded migrations: %w", err)
	}

	migrations := make([]Migration, 0, len(paths))
	seenVersions := make(map[int64]string, len(paths))

	for _, path := range paths {
		filename := filepath.Base(path)
		matches := migrationNamePattern.FindStringSubmatch(filename)
		if matches == nil {
			return nil, fmt.Errorf(
				"invalid migration filename %q; expected NNNNNN_name.up.sql",
				filename,
			)
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", filename, err)
		}
		if previous, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf(
				"duplicate migration version %d in %q and %q",
				version,
				previous,
				filename,
			)
		}

		body, err := migrationFiles.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", filename, err)
		}
		sql := strings.TrimSpace(string(body))
		if sql == "" {
			return nil, fmt.Errorf("migration %q is empty", filename)
		}

		sum := sha256.Sum256(body)
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     matches[2],
			Filename: filename,
			SQL:      sql,
			Checksum: hex.EncodeToString(sum[:]),
		})
		seenVersions[version] = filename
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func (r *Runner) Apply(ctx context.Context, dryRun bool) error {
	migrations, err := Load()
	if err != nil {
		return err
	}

	if dryRun {
		statuses, err := r.Status(ctx)
		if err != nil {
			return err
		}
		for _, status := range statuses {
			if !status.Applied {
				fmt.Fprintf(
					r.out,
					"would apply %06d_%s\n",
					status.Migration.Version,
					status.Migration.Name,
				)
			}
		}
		return nil
	}

	if err := ensureSchemaMigrationsTable(ctx, r.db); err != nil {
		return err
	}

	applied, err := loadApplied(ctx, r.db)
	if err != nil {
		return err
	}
	if err := verifyChecksums(migrations, applied); err != nil {
		return err
	}

	for _, migration := range migrations {
		if _, exists := applied[migration.Version]; exists {
			continue
		}

		err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(migration.SQL).Error; err != nil {
				return fmt.Errorf("execute %s: %w", migration.Filename, err)
			}
			if err := tx.Exec(
				`INSERT INTO schema_migrations (version, name, checksum, applied_at)
				 VALUES (?, ?, ?, ?)`,
				migration.Version,
				migration.Name,
				migration.Checksum,
				time.Now().UTC(),
			).Error; err != nil {
				return fmt.Errorf("record %s: %w", migration.Filename, err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(
			r.out,
			"applied %06d_%s\n",
			migration.Version,
			migration.Name,
		)
	}
	return nil
}

func (r *Runner) Verify(ctx context.Context) error {
	migrations, err := Load()
	if err != nil {
		return err
	}

	if !r.db.Migrator().HasTable("schema_migrations") {
		return errors.New("schema_migrations table does not exist; run migrations first")
	}

	applied, err := loadApplied(ctx, r.db)
	if err != nil {
		return err
	}
	if err := verifyChecksums(migrations, applied); err != nil {
		return err
	}

	known := make(map[int64]struct{}, len(migrations))
	for _, migration := range migrations {
		known[migration.Version] = struct{}{}
	}
	for version := range applied {
		if _, exists := known[version]; !exists {
			return fmt.Errorf("database contains unknown migration version %d", version)
		}
	}

	fmt.Fprintf(r.out, "verified %d applied migration(s)\n", len(applied))
	return nil
}

func (r *Runner) Status(ctx context.Context) ([]Status, error) {
	migrations, err := Load()
	if err != nil {
		return nil, err
	}

	applied := map[int64]AppliedMigration{}
	if r.db.Migrator().HasTable("schema_migrations") {
		applied, err = loadApplied(ctx, r.db)
		if err != nil {
			return nil, err
		}
		if err := verifyChecksums(migrations, applied); err != nil {
			return nil, err
		}
	}

	statuses := make([]Status, 0, len(migrations))
	for _, migration := range migrations {
		status := Status{Migration: migration}
		if row, exists := applied[migration.Version]; exists {
			status.Applied = true
			appliedAt := row.AppliedAt
			status.AppliedAt = &appliedAt
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func ensureSchemaMigrationsTable(ctx context.Context, db *gorm.DB) error {
	const statement = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version BIGINT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	checksum VARCHAR(64) NOT NULL,
	applied_at TIMESTAMP NOT NULL
)`
	if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func loadApplied(
	ctx context.Context,
	db *gorm.DB,
) (map[int64]AppliedMigration, error) {
	var rows []AppliedMigration
	if err := db.WithContext(ctx).
		Raw(`SELECT version, name, checksum, applied_at
		     FROM schema_migrations
		     ORDER BY version`).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load applied migrations: %w", err)
	}

	applied := make(map[int64]AppliedMigration, len(rows))
	for _, row := range rows {
		applied[row.Version] = row
	}
	return applied, nil
}

func verifyChecksums(
	migrations []Migration,
	applied map[int64]AppliedMigration,
) error {
	for _, migration := range migrations {
		row, exists := applied[migration.Version]
		if !exists {
			continue
		}
		if row.Checksum != migration.Checksum {
			return fmt.Errorf(
				"checksum mismatch for migration %s; applied migrations are immutable",
				migration.Filename,
			)
		}
		if row.Name != migration.Name {
			return fmt.Errorf(
				"name mismatch for migration version %d: database=%q source=%q",
				migration.Version,
				row.Name,
				migration.Name,
			)
		}
	}
	return nil
}
