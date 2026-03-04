package db

import (
	"context"
	"testing"
	"time"

	"github.com/dbehnke/trindex/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		DatabaseURL:       "postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable",
		DBMaxConns:        10,
		DBMinConns:        2,
		DBMaxConnLifetime: 60,
		DBMaxConnIdleTime: 30,
	}
}

func TestNew(t *testing.T) {
	cfg := testConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return
	}
	defer db.Close()

	if db.Pool() == nil {
		t.Error("expected pool to be initialized")
	}

	err = db.Pool().Ping(ctx)
	if err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestDB_Migrate(t *testing.T) {
	cfg := testConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return
	}
	defer db.Close()

	err = db.Migrate(ctx)
	if err != nil {
		t.Errorf("migrate failed: %v", err)
	}

	var tableExists bool
	query := `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'memories')`
	err = db.Pool().QueryRow(ctx, query).Scan(&tableExists)
	if err != nil {
		t.Errorf("failed to check table existence: %v", err)
	}
	if !tableExists {
		t.Error("expected memories table to exist after migration")
	}
}

func TestDB_ExtensionExists(t *testing.T) {
	cfg := testConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return
	}
	defer db.Close()

	err = db.Migrate(ctx)
	if err != nil {
		t.Skipf("Migration failed: %v", err)
		return
	}

	extensions := []string{"vector", "pg_trgm"}
	for _, ext := range extensions {
		var exists bool
		query := `SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = $1)`
		err := db.Pool().QueryRow(ctx, query, ext).Scan(&exists)
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
	cfg := testConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return
	}
	defer db.Close()

	err = db.Migrate(ctx)
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
		err := db.Pool().QueryRow(ctx, query, idx).Scan(&exists)
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
	cfg := testConfig()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return
	}

	db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = db.Pool().Ping(ctx)
	if err == nil {
		t.Error("expected ping to fail after close")
	}
}

func TestDB_Pool(t *testing.T) {
	cfg := testConfig()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return
	}
	defer db.Close()

	pool := db.Pool()
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
	cfg := testConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return
	}
	defer db.Close()

	err = db.Health(ctx)
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
}
