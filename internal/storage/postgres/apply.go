package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const migrationAdvisoryLockKey int64 = 0x47434665656473

// Migrate applies all embedded migrations transactionally.
//
// One PostgreSQL transaction owns the migration ledger update and an advisory
// transaction lock serializes concurrent migration attempts. Existing ledger
// rows are verified by migration name and SHA-256 checksum before they are
// accepted as already applied.
func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("postgres store is not open")
	}

	migrations, err := Migrations()
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", migrationAdvisoryLockKey); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}

	if _, err := tx.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS goreecloud_feeds"); err != nil {
		return fmt.Errorf("ensure application schema: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS goreecloud_feeds.schema_migrations (
			version integer PRIMARY KEY CHECK (version > 0),
			name text NOT NULL UNIQUE,
			checksum char(64) NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
			applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("ensure migration ledger: %w", err)
	}

	for _, migration := range migrations {
		checksum := migrationChecksum(migration.SQL)

		var appliedName string
		var appliedChecksum string
		err := tx.QueryRow(
			ctx,
			"SELECT name, checksum FROM goreecloud_feeds.schema_migrations WHERE version = $1",
			migration.Version,
		).Scan(&appliedName, &appliedChecksum)

		switch {
		case err == nil:
			if appliedName != migration.Name || appliedChecksum != checksum {
				return fmt.Errorf(
					"migration %04d integrity mismatch: recorded %q/%s, source %q/%s",
					migration.Version,
					appliedName,
					appliedChecksum,
					migration.Name,
					checksum,
				)
			}
			continue
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("read migration ledger for %04d: %w", migration.Version, err)
		}

		body, err := migrationBody(migration.SQL)
		if err != nil {
			return fmt.Errorf("prepare migration %s: %w", migration.Name, err)
		}

		if _, err := tx.Conn().PgConn().Exec(ctx, body).ReadAll(); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}
		if _, err := tx.Exec(
			ctx,
			"INSERT INTO goreecloud_feeds.schema_migrations (version, name, checksum) VALUES ($1, $2, $3)",
			migration.Version,
			migration.Name,
			checksum,
		); err != nil {
			return fmt.Errorf("record migration %s: %w", migration.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

func migrationChecksum(sql string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(sql)))
	return hex.EncodeToString(sum[:])
}

func migrationBody(sql string) (string, error) {
	trimmed := strings.TrimSpace(sql)
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "BEGIN;") {
		return "", fmt.Errorf("migration must begin with BEGIN;")
	}
	if !strings.HasSuffix(upper, "COMMIT;") {
		return "", fmt.Errorf("migration must end with COMMIT;")
	}

	body := strings.TrimSpace(trimmed[len("BEGIN;") : len(trimmed)-len("COMMIT;")])
	if body == "" {
		return "", fmt.Errorf("migration body is empty")
	}
	return body, nil
}
