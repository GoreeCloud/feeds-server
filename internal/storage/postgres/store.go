package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultConnectTimeout = 5 * time.Second
	defaultMaxConns       = int32(4)
)

// Store owns the GoreeCloud Feeds PostgreSQL connection pool.
//
// Constructing a Store does not apply migrations. Call Migrate explicitly after
// opening the database so schema changes remain visible and separately handled.
type Store struct {
	pool *pgxpool.Pool
}

// Open creates and verifies a bounded PostgreSQL connection pool.
//
// The connection string is never included in returned errors because it may
// contain credentials. Production connection policy and secret injection remain
// deployment concerns outside this Development tranche.
func Open(ctx context.Context, connectionString string) (*Store, error) {
	if strings.TrimSpace(connectionString) == "" {
		return nil, fmt.Errorf("postgres connection string is required")
	}

	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("parse postgres connection configuration: %w", err)
	}

	config.MaxConns = defaultMaxConns
	config.MinConns = 0
	config.ConnConfig.ConnectTimeout = defaultConnectTimeout
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["application_name"] = "goreecloud-feeds-server"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	store := &Store{pool: pool}
	pingCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()
	if err := store.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	return store, nil
}

// Ping verifies that PostgreSQL is reachable through the current pool.
func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("postgres store is not open")
	}
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

// Close releases the PostgreSQL pool.
func (s *Store) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}
