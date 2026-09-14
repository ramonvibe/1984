package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ramon/trackline/db/migrations"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	// Serialize startup across server instances and apply the migration batch atomically.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(1984000)"); err != nil { return err }
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			checksum text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version, err := strconv.ParseInt(strings.SplitN(name, "_", 2)[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid migration %q", name)
		}
		body, err := migrations.Files.ReadFile(name)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(body)
		checksum := hex.EncodeToString(sum[:])

		var stored string
		err = tx.QueryRow(ctx, "SELECT checksum FROM schema_migrations WHERE version = $1", version).Scan(&stored)
		if err == nil {
			if stored != checksum {
				return fmt.Errorf("migration %d changed after being applied", version)
			}
			continue
		}

		if !errors.Is(err, pgx.ErrNoRows) { return err }
		if _, err = tx.Exec(ctx, string(body)); err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)", version, checksum)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return tx.Commit(ctx)
}
