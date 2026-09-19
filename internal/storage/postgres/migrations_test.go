package postgres

import (
	"strings"
	"testing"
)

func TestMigrationsAreOrderedAndContiguous(t *testing.T) {
	migrations, err := Migrations()
	if err != nil {
		t.Fatalf("Migrations() error = %v", err)
	}
	if len(migrations) != 1 {
		t.Fatalf("expected 1 migration, got %d", len(migrations))
	}
	if migrations[0].Version != 1 || migrations[0].Name != "0001_initial.sql" {
		t.Fatalf("unexpected first migration: %+v", migrations[0])
	}
}

func TestInitialMigrationDefinesStorageBoundary(t *testing.T) {
	migrations, err := Migrations()
	if err != nil {
		t.Fatalf("Migrations() error = %v", err)
	}
	sql := migrations[0].SQL

	required := []string{
		"CREATE SCHEMA IF NOT EXISTS goreecloud_feeds",
		"CREATE TABLE goreecloud_feeds.users",
		"CREATE TABLE goreecloud_feeds.feeds",
		"CREATE TABLE goreecloud_feeds.subscriptions",
		"CREATE TABLE goreecloud_feeds.articles",
		"CREATE TABLE goreecloud_feeds.article_identity_keys",
		"CREATE TABLE goreecloud_feeds.article_source_history",
		"CREATE TABLE goreecloud_feeds.article_states",
		"identity_subject text NOT NULL UNIQUE",
		"preserved boolean NOT NULL DEFAULT false",
		"read_position >= 0 AND read_position <= 1",
		"FOREIGN KEY (article_id, feed_id)",
		"key_type IN ('source_identifier', 'normalized_url', 'content_fingerprint')",
	}
	for _, token := range required {
		if !strings.Contains(sql, token) {
			t.Fatalf("initial migration missing %q", token)
		}
	}
}

func TestInitialMigrationKeepsArticleContentUserIndependent(t *testing.T) {
	migrations, err := Migrations()
	if err != nil {
		t.Fatalf("Migrations() error = %v", err)
	}
	sql := migrations[0].SQL

	start := strings.Index(sql, "CREATE TABLE goreecloud_feeds.articles")
	if start < 0 {
		t.Fatal("articles table not found")
	}
	end := strings.Index(sql[start:], "\n);")
	if end < 0 {
		t.Fatal("articles table end not found")
	}
	articlesTable := sql[start : start+end]

	if strings.Contains(articlesTable, "user_id") {
		t.Fatal("shared article content must not contain user-owned state")
	}
}

func TestInitialMigrationIsAdditiveFoundation(t *testing.T) {
	migrations, err := Migrations()
	if err != nil {
		t.Fatalf("Migrations() error = %v", err)
	}
	upper := strings.ToUpper(migrations[0].SQL)

	for _, destructive := range []string{
		"DROP TABLE",
		"DROP SCHEMA",
		"TRUNCATE ",
		"DELETE FROM ",
		"ALTER TABLE ",
		"CREATE EXTENSION",
	} {
		if strings.Contains(upper, destructive) {
			t.Fatalf("initial migration must remain additive and extension-free; found %q", destructive)
		}
	}

	if !strings.HasPrefix(strings.TrimSpace(upper), "BEGIN;") {
		t.Fatal("migration must begin in an explicit transaction")
	}
	if !strings.HasSuffix(strings.TrimSpace(upper), "COMMIT;") {
		t.Fatal("migration must end with COMMIT")
	}
}

func TestMigrationNameValidation(t *testing.T) {
	valid := []string{"0001_initial.sql", "0042_article_state.sql"}
	for _, name := range valid {
		if _, err := migrationVersion(name); err != nil {
			t.Fatalf("migrationVersion(%q) unexpected error: %v", name, err)
		}
	}

	invalid := []string{
		"1_initial.sql",
		"0000_initial.sql",
		"0001-Initial.sql",
		"0001_.sql",
		"0001_initial.txt",
	}
	for _, name := range invalid {
		if _, err := migrationVersion(name); err == nil {
			t.Fatalf("migrationVersion(%q) expected error", name)
		}
	}
}
