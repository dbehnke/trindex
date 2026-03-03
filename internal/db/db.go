package db

import (
	"context"
	"fmt"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a PostgreSQL connection pool
type DB struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// New creates a new database connection pool
func New(cfg *config.Config) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		pool: pool,
		cfg:  cfg,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// Pool returns the underlying connection pool
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Health checks if the database is reachable
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.pool.Ping(ctx)
}
