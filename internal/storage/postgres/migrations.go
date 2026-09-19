package postgres

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Migration struct {
	Version int
	Name    string
	SQL     string
}

// Migrations returns the ordered, embedded PostgreSQL schema migrations.
//
// Execution against a live database is intentionally outside this Development
// schema-foundation tranche. A later database-connectivity layer will apply
// these migrations transactionally and record applied versions.
func Migrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))
	versions := make(map[int]string)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, err := migrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		if existing, ok := versions[version]; ok {
			return nil, fmt.Errorf("duplicate migration version %04d: %s and %s", version, existing, entry.Name())
		}

		body, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		sql := strings.TrimSpace(string(body))
		if sql == "" {
			return nil, fmt.Errorf("migration %s is empty", entry.Name())
		}

		versions[version] = entry.Name()
		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     sql,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	for i, migration := range migrations {
		expected := i + 1
		if migration.Version != expected {
			return nil, fmt.Errorf(
				"migration sequence must be contiguous: expected %04d, found %04d (%s)",
				expected,
				migration.Version,
				migration.Name,
			)
		}
	}

	return migrations, nil
}

func migrationVersion(name string) (int, error) {
	if len(name) < 9 || name[4] != '_' || !strings.HasSuffix(name, ".sql") {
		return 0, fmt.Errorf("invalid migration name %q", name)
	}

	for _, ch := range name[:4] {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid migration version in %q", name)
		}
	}

	slug := strings.TrimSuffix(name[5:], ".sql")
	if slug == "" {
		return 0, fmt.Errorf("migration name %q has an empty subject", name)
	}
	for _, ch := range slug {
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '_' {
			return 0, fmt.Errorf("migration name %q must use lowercase letters, digits, and underscores", name)
		}
	}

	version, err := strconv.Atoi(name[:4])
	if err != nil || version < 1 {
		return 0, fmt.Errorf("invalid migration version in %q", name)
	}
	return version, nil
}
