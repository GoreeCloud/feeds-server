package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitialMigrationHasRequiredStorageBoundaries(t *testing.T) {
	migration := readInitialMigration(t)
	upper := strings.ToUpper(migration)

	requiredTables := []string{
		"CREATE TABLE FEEDS",
		"CREATE TABLE SUBSCRIPTIONS",
		"CREATE TABLE ARTICLES",
		"CREATE TABLE ARTICLE_ALIASES",
		"CREATE TABLE ARTICLE_SOURCE_HISTORY",
		"CREATE TABLE ARTICLE_STATES",
		"CREATE TABLE ARTICLE_CATEGORIES",
		"CREATE TABLE ARTICLE_TAGS",
		"CREATE TABLE ARTICLE_MEDIA",
		"CREATE TABLE ARTICLE_IMAGES",
	}

	for _, table := range requiredTables {
		if !strings.Contains(upper, table) {
			t.Fatalf("initial migration missing %s", table)
		}
	}

	requiredFragments := []string{
		"PRESERVED_AT TIMESTAMPTZ",
		"CONTENT_FINGERPRINT TEXT",
		"FIRST_RETRIEVED_AT TIMESTAMPTZ",
		"LAST_RETRIEVED_AT TIMESTAMPTZ",
		"RETENTION_DAYS INTEGER",
		"PRIMARY KEY (FEED_ID, ALIAS_TYPE, ALIAS_VALUE)",
		"PRIMARY KEY (USER_ID, ARTICLE_ID)",
		"REFERENCES ARTICLES(ID) ON DELETE CASCADE",
		"REFERENCES FEEDS(ID) ON DELETE CASCADE",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(upper, fragment) {
			t.Fatalf("initial migration missing required storage fragment %q", fragment)
		}
	}
}

func TestInitialMigrationIsAdditiveAndApplicationIDOwned(t *testing.T) {
	migration := readInitialMigration(t)
	upper := strings.ToUpper(migration)

	for _, forbidden := range []string{
		"DROP TABLE",
		"DROP COLUMN",
		"TRUNCATE ",
		"BIGSERIAL",
		"SERIAL ",
		"GENERATED ALWAYS AS IDENTITY",
	} {
		if strings.Contains(upper, forbidden) {
			t.Fatalf("initial additive migration contains forbidden fragment %q", forbidden)
		}
	}

	if !strings.HasPrefix(strings.TrimSpace(upper), "BEGIN;") {
		t.Fatal("initial migration must begin with BEGIN")
	}
	if !strings.HasSuffix(strings.TrimSpace(upper), "COMMIT;") {
		t.Fatal("initial migration must end with COMMIT")
	}
}

func TestInitialMigrationKeepsIdentityAndContentStateSeparated(t *testing.T) {
	migration := readInitialMigration(t)
	upper := strings.ToUpper(migration)

	if strings.Contains(upper, "CREATE TABLE USERS") {
		t.Fatal("Feeds must not create a competing local authentication user table in the initial persistence schema")
	}
	if !strings.Contains(upper, "USER_ID TEXT NOT NULL") {
		t.Fatal("user-owned state must retain an external identity reference")
	}
	if !strings.Contains(upper, "CREATE TABLE ARTICLE_STATES") {
		t.Fatal("user article state must remain separate from canonical article content")
	}
}

func readInitialMigration(t *testing.T) string {
	t.Helper()

	path := filepath.Join("..", "..", "migrations", "0001_initial.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read initial migration: %v", err)
	}
	return string(data)
}
