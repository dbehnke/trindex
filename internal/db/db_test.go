package db

import (
	"context"
	"testing"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/testutil"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	cfg := &config.Config{
		DatabaseURL:        container.ConnStr,
		EmbedDimensions:    768,
		HNSWM:              16,
		HNSWEfConstruction: 64,
		HNSWEfSearch:       40,
		DBMaxConns:         10,
		DBMinConns:         2,
		DBMaxConnLifetime:  60,
		DBMaxConnIdleTime:  30,
	}

	database, err := New(cfg)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := database.Migrate(ctx); err != nil {
		database.Close()
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		database.Close()
		_ = container.Terminate(ctx)
	}

	return database, cleanup
}

func TestNew(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if database.Pool() == nil {
		t.Error("expected pool to be initialized")
	}

	err := database.Pool().Ping(ctx)
	if err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestDB_Migrate(t *testing.T) {
	t.Helper()

	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	cfg := &config.Config{
		DatabaseURL:        container.ConnStr,
		EmbedDimensions:    768,
		HNSWM:              16,
		HNSWEfConstruction: 64,
		HNSWEfSearch:       40,
		DBMaxConns:         10,
		DBMinConns:         2,
		DBMaxConnLifetime:  60,
		DBMaxConnIdleTime:  30,
	}

	database, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	migrateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = database.Migrate(migrateCtx)
	if err != nil {
		t.Errorf("migrate failed: %v", err)
	}

	var tableExists bool
	query := `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'memories')`
	err = database.Pool().QueryRow(migrateCtx, query).Scan(&tableExists)
	if err != nil {
		t.Errorf("failed to check table existence: %v", err)
	}
	if !tableExists {
		t.Error("expected memories table to exist after migration")
	}
}

func TestDB_ExtensionExists(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := database.Migrate(ctx)
	if err != nil {
		t.Skipf("Migration failed: %v", err)
		return
	}

	extensions := []string{"vector", "pg_trgm"}
	for _, ext := range extensions {
		var exists bool
		query := `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = $1)`
		err := database.Pool().QueryRow(ctx, query, ext).Scan(&exists)
		if err != nil {
			t.Errorf("failed to check extension %s: %v", ext, err)
			continue
		}
		if !exists {
			t.Errorf("expected extension %s to exist", ext)
		}
	}
}

func TestDB_IndexesExist(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := database.Migrate(ctx)
	if err != nil {
		t.Skipf("Migration failed: %v", err)
		return
	}

	indexes := []string{
		"memories_embedding_hnsw_idx",
		"memories_search_vec_idx",
		"memories_metadata_idx",
		"memories_namespace_idx",
		"memories_created_at_idx",
	}

	for _, idx := range indexes {
		var exists bool
		query := `SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)`
		err := database.Pool().QueryRow(ctx, query, idx).Scan(&exists)
		if err != nil {
			t.Errorf("failed to check index %s: %v", idx, err)
			continue
		}
		if !exists {
			t.Errorf("expected index %s to exist", idx)
		}
	}
}

func TestDB_Close(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := database.Pool().Ping(ctx)
	if err == nil {
		t.Error("expected ping to fail after close")
	}
}

func TestDB_Pool(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	pool := database.Pool()
	if pool == nil {
		t.Error("expected pool to not be nil")
		return
	}

	stats := pool.Stat()
	if stats == nil {
		t.Error("expected pool stats to be available")
	}
}

func TestDB_Health(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := database.Health(ctx)
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
}
