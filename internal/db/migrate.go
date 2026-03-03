package db

import (
	"context"
	"fmt"
)

// Migrate runs database schema migrations
func (db *DB) Migrate(ctx context.Context) error {
	// Enable required extensions
	if err := db.enableExtensions(ctx); err != nil {
		return fmt.Errorf("failed to enable extensions: %w", err)
	}

	// Create memories table
	if err := db.createMemoriesTable(ctx); err != nil {
		return fmt.Errorf("failed to create memories table: %w", err)
	}

	// Create indexes
	if err := db.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Create updated_at trigger
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
	// Create function
	_, err := db.pool.Exec(ctx, `
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql`)
	if err != nil {
		return fmt.Errorf("failed to create function: %w", err)
	}

	// Create trigger
	_, err = db.pool.Exec(ctx, `
CREATE TRIGGER memories_updated_at
    BEFORE UPDATE ON memories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at()`)
	if err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	return nil
}
