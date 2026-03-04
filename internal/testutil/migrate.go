package testutil

import (
	"context"
	"fmt"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupTestDB creates a database connection pool and runs migrations.
func SetupTestDB(ctx context.Context, connStr string, embeddingDims int) (*db.DB, error) {
	cfg := &config.Config{
		DatabaseURL:        connStr,
		EmbedDimensions:    embeddingDims,
		HNSWM:              16,
		HNSWEfConstruction: 64,
		HNSWEfSearch:       40,
		DBMaxConns:         10,
		DBMinConns:         2,
		DBMaxConnLifetime:  60,
		DBMaxConnIdleTime:  30,
	}

	database, err := db.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	if err := database.Migrate(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// TruncateTables truncates all tables in the database for clean test state.
func TruncateTables(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, "TRUNCATE TABLE memories RESTART IDENTITY CASCADE")
	if err != nil {
		return fmt.Errorf("failed to truncate memories table: %w", err)
	}
	return nil
}
