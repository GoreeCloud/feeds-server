package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMigrationBodyRequiresTransactionWrapper(t *testing.T) {
	body, err := migrationBody("BEGIN;\nSELECT 1;\nCOMMIT;")
	if err != nil {
		t.Fatalf("migrationBody() error = %v", err)
	}
	if strings.TrimSpace(body) != "SELECT 1;" {
		t.Fatalf("unexpected migration body %q", body)
	}

	for _, input := range []string{
		"SELECT 1;",
		"BEGIN; SELECT 1;",
		"SELECT 1; COMMIT;",
		"BEGIN; COMMIT;",
	} {
		if _, err := migrationBody(input); err == nil {
			t.Fatalf("migrationBody(%q) expected error", input)
		}
	}
}

func TestMigrationChecksumIsStable(t *testing.T) {
	first := migrationChecksum("  SELECT 1;\n")
	second := migrationChecksum("SELECT 1;")
	if first != second {
		t.Fatalf("trim-equivalent migration checksums differ: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("expected SHA-256 hex checksum, got %d characters", len(first))
	}
}

func TestOpenRejectsBlankConnectionString(t *testing.T) {
	if _, err := Open(context.Background(), "   "); err == nil {
		t.Fatal("Open() expected blank connection string error")
	}
}

func TestPostgresConnectivityAndMigrations(t *testing.T) {
	connectionString := os.Getenv("FEEDS_TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("FEEDS_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := Open(ctx, connectionString)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("first Migrate() error = %v", err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate() must be idempotent: %v", err)
	}

	var applicationName string
	if err := store.pool.QueryRow(ctx, "SELECT current_setting('application_name')").Scan(&applicationName); err != nil {
		t.Fatalf("read application_name: %v", err)
	}
	if applicationName != "goreecloud-feeds-server" {
		t.Fatalf("unexpected application_name %q", applicationName)
	}

	var articleTableExists bool
	if err := store.pool.QueryRow(
		ctx,
		"SELECT to_regclass('goreecloud_feeds.articles') IS NOT NULL",
	).Scan(&articleTableExists); err != nil {
		t.Fatalf("verify articles table: %v", err)
	}
	if !articleTableExists {
		t.Fatal("expected goreecloud_feeds.articles after migration")
	}

	var ledgerCount int
	if err := store.pool.QueryRow(
		ctx,
		"SELECT count(*) FROM goreecloud_feeds.schema_migrations",
	).Scan(&ledgerCount); err != nil {
		t.Fatalf("count migration ledger: %v", err)
	}
	if ledgerCount != 1 {
		t.Fatalf("expected 1 applied migration, got %d", ledgerCount)
	}

	var ledgerMatches bool
	migrations, err := Migrations()
	if err != nil {
		t.Fatalf("Migrations() error = %v", err)
	}
	expected := migrations[0]
	if err := store.pool.QueryRow(
		ctx,
		`SELECT name = $1 AND checksum = $2
		 FROM goreecloud_feeds.schema_migrations
		 WHERE version = $3`,
		expected.Name,
		migrationChecksum(expected.SQL),
		expected.Version,
	).Scan(&ledgerMatches); err != nil {
		t.Fatalf("verify migration ledger: %v", err)
	}
	if !ledgerMatches {
		t.Fatal("migration ledger does not match embedded source")
	}
}
