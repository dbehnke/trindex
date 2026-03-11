package db

import (
	"context"
	"fmt"
)

// Migrate runs database schema migrations
func (db *DB) Migrate(ctx context.Context) error {
	if err := db.enableExtensions(ctx); err != nil {
		return fmt.Errorf("failed to enable extensions: %w", err)
	}
	if err := db.createAPIKeysTable(ctx); err != nil {
		return fmt.Errorf("failed to create api_keys table: %w", err)
	}
	if err := db.createAuditLogsTable(ctx); err != nil {
		return fmt.Errorf("failed to create audit_logs table: %w", err)
	}
	if err := db.createMemoriesTable(ctx); err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}
	if err := db.migrateMemoriesV2(ctx); err != nil {
		return fmt.Errorf("failed to migrate memories v2: %w", err)
	}
	if err := db.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	if err := db.createMemoriesV2Indexes(ctx); err != nil {
		return fmt.Errorf("failed to create memories v2 indexes: %w", err)
	}
	if err := db.createUpdatedAtTrigger(ctx); err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	return nil
}

func (db *DB) enableExtensions(ctx context.Context) error {
	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS vector",
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
	}

	for _, ext := range extensions {
		if _, err := db.pool.Exec(ctx, ext); err != nil {
			return fmt.Errorf("failed to execute %q: %w", ext, err)
		}
	}

	return nil
}

func (db *DB) createAPIKeysTable(ctx context.Context) error {
	query := `
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE
)`
	_, err := db.pool.Exec(ctx, query)
	return err
}

func (db *DB) createAuditLogsTable(ctx context.Context) error {
	query := `
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    api_key_id UUID REFERENCES api_keys(id),
    action VARCHAR(50) NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    details JSONB DEFAULT '{}'
)`
	_, err := db.pool.Exec(ctx, query)
	return err
}

func (db *DB) createMemoriesTable(ctx context.Context) error {
	dims := db.cfg.EmbedDimensions
	query := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS memories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace TEXT NOT NULL DEFAULT 'default',
    content TEXT NOT NULL,
    embedding VECTOR(%d),
    metadata JSONB DEFAULT '{}',
    search_vec TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', content)) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`, dims)

	_, err := db.pool.Exec(ctx, query)
	return err
}

func (db *DB) createIndexes(ctx context.Context) error {
	indexes := []string{
		// HNSW vector index (cosine distance)
		fmt.Sprintf(`
CREATE INDEX IF NOT EXISTS memories_embedding_hnsw_idx
    ON memories
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = %d, ef_construction = %d)`,
			db.cfg.HNSWM, db.cfg.HNSWEfConstruction),

		// Full-text search index
		`CREATE INDEX IF NOT EXISTS memories_search_vec_idx
    ON memories USING gin(search_vec)`,

		// JSONB metadata index
		`CREATE INDEX IF NOT EXISTS memories_metadata_idx
    ON memories USING gin(metadata)`,

		// Namespace index
		`CREATE INDEX IF NOT EXISTS memories_namespace_idx
    ON memories (namespace)`,

		// Timestamp index
		`CREATE INDEX IF NOT EXISTS memories_created_at_idx
    ON memories (created_at DESC)`,
	}

	for _, idx := range indexes {
		if _, err := db.pool.Exec(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (db *DB) createUpdatedAtTrigger(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, `
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql`)
	if err != nil {
		return fmt.Errorf("failed to create function: %w", err)
	}

	_, err = db.pool.Exec(ctx, `DROP TRIGGER IF EXISTS memories_updated_at ON memories;`)
	if err != nil {
		return fmt.Errorf("failed to drop existing trigger: %w", err)
	}

	_, err = db.pool.Exec(ctx, `
CREATE TRIGGER memories_updated_at
    BEFORE UPDATE ON memories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at()`)
	if err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	return nil
}

func (db *DB) migrateMemoriesV2(ctx context.Context) error {
	queries := []string{
		`ALTER TABLE memories ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64)`,
		`ALTER TABLE memories ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ`,
		`ALTER TABLE memories ADD COLUMN IF NOT EXISTS ttl_seconds INTEGER`,
	}

	for _, q := range queries {
		if _, err := db.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed to execute %q: %w", q, err)
		}
	}

	return nil
}

func (db *DB) createMemoriesV2Indexes(ctx context.Context) error {
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS memories_content_hash_unique_idx
			ON memories (namespace, content_hash)
			WHERE content_hash IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS memories_expires_at_idx
			ON memories (expires_at)
			WHERE expires_at IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS memories_ttl_seconds_idx
			ON memories (ttl_seconds)
			WHERE ttl_seconds IS NOT NULL`,
	}

	for _, idx := range indexes {
		if _, err := db.pool.Exec(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
