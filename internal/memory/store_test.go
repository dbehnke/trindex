package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/google/uuid"
)

func setupTestStore(t *testing.T) (*Store, *db.DB, func()) {
	cfg := &config.Config{
		DatabaseURL:                "postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable",
		EmbedBaseURL:               "http://localhost:11434/v1",
		EmbedModel:                 "nomic-embed-text",
		EmbedAPIKey:                "ollama",
		EmbedDimensions:            768,
		HNSWM:                      16,
		HNSWEfConstruction:         64,
		HNSWEfSearch:               40,
		DefaultNamespace:           "default",
		DefaultTopK:                10,
		DefaultSimilarityThreshold: 0.7,
		DBMaxConns:                 10,
		DBMinConns:                 2,
		DBMaxConnLifetime:          60,
		DBMaxConnIdleTime:          30,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return nil, nil, func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = database.Migrate(ctx)
	if err != nil {
		database.Close()
		t.Skipf("Migration failed: %v", err)
		return nil, nil, func() {}
	}

	embedClient := embed.NewClient(cfg)
	store := NewStore(database, embedClient, cfg)

	cleanup := func() {
		database.Close()
	}

	return store, database, cleanup
}

func TestStore_Create(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	memory, err := store.Create(ctx, "Test memory content", "test-namespace", map[string]interface{}{
		"tags": []string{"test", "memory"},
	})
	if err != nil {
		t.Skipf("Embedding service not available: %v", err)
		return
	}

	if memory.ID.String() == "" {
		t.Error("expected memory ID to be set")
	}
	if memory.Content != "Test memory content" {
		t.Errorf("expected content 'Test memory content', got '%s'", memory.Content)
	}
	if memory.Namespace != "test-namespace" {
		t.Errorf("expected namespace 'test-namespace', got '%s'", memory.Namespace)
	}
	if memory.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestStore_Create_DefaultNamespace(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	memory, err := store.Create(ctx, "Test content", "", nil)
	if err != nil {
		t.Skipf("Embedding service not available: %v", err)
		return
	}

	if memory.Namespace != "default" {
		t.Errorf("expected default namespace, got '%s'", memory.Namespace)
	}
}

func TestStore_GetByID(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := store.Create(ctx, "Test memory for get", "test", nil)
	if err != nil {
		t.Skipf("Embedding service not available: %v", err)
		return
	}

	retrieved, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Errorf("failed to get memory: %v", err)
		return
	}

	if retrieved.ID != created.ID {
		t.Error("retrieved memory ID doesn't match")
	}
	if retrieved.Content != created.Content {
		t.Error("retrieved memory content doesn't match")
	}
}

func TestStore_GetByID_NotFound(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.GetByID(ctx, uuid.New())
	if err == nil {
		t.Error("expected error for non-existent memory")
	}
}

func TestStore_List(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Memory 1", "list-test", nil)
	if err != nil {
		t.Skipf("Embedding service not available: %v", err)
		return
	}
	_, _ = store.Create(ctx, "Memory 2", "list-test", nil)

	params := ListParams{
		Namespace: "list-test",
		Limit:     10,
		Offset:    0,
		Order:     "desc",
	}

	memories, err := store.List(ctx, params)
	if err != nil {
		t.Errorf("failed to list memories: %v", err)
		return
	}

	if len(memories) < 1 {
		t.Errorf("expected at least 1 memory, got %d", len(memories))
	}
}

func TestStore_DeleteByID(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := store.Create(ctx, "Memory to delete", "test", nil)
	if err != nil {
		t.Skipf("Embedding service not available: %v", err)
		return
	}

	err = store.DeleteByID(ctx, created.ID)
	if err != nil {
		t.Errorf("failed to delete memory: %v", err)
	}

	_, err = store.GetByID(ctx, created.ID)
	if err == nil {
		t.Error("expected memory to be deleted")
	}
}

func TestStore_DeleteByNamespace(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, _ = store.Create(ctx, "Memory 1", "delete-ns", nil)
	_, _ = store.Create(ctx, "Memory 2", "delete-ns", nil)

	filter := ForgetFilter{}
	count, err := store.DeleteByNamespace(ctx, "delete-ns", filter)
	if err != nil {
		t.Errorf("failed to delete by namespace: %v", err)
		return
	}

	if count < 1 {
		t.Errorf("expected at least 1 memory deleted, got %d", count)
	}
}

func TestStore_GetStats(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Stats test memory", "stats-test", nil)
	if err != nil {
		t.Skipf("Embedding service not available: %v", err)
		return
	}

	stats, err := store.GetStats(ctx, "")
	if err != nil {
		t.Errorf("failed to get stats: %v", err)
		return
	}

	if stats.TotalMemories < 1 {
		t.Error("expected at least 1 total memory")
	}
	if stats.ByNamespace == nil {
		t.Error("expected ByNamespace to be initialized")
	}
	if stats.EmbeddingModel == "" {
		t.Error("expected EmbeddingModel to be set")
	}
}

func TestStore_Recall(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "The quick brown fox jumps over the lazy dog", "recall-test", nil)
	if err != nil {
		t.Skipf("Embedding service not available: %v", err)
		return
	}

	params := RecallParams{
		Query:      "quick fox",
		Namespaces: []string{"recall-test"},
		TopK:       5,
		Threshold:  0.5,
		Filter:     Filter{},
	}

	results, err := store.Recall(ctx, params)
	if err != nil {
		t.Skipf("Recall failed (embedding may not be available): %v", err)
		return
	}

	if len(results) < 1 {
		t.Errorf("expected at least 1 result, got %d", len(results))
	}

	for _, r := range results {
		if r.Score < 0 {
			t.Error("expected score to be non-negative")
		}
		if r.Content == "" {
			t.Error("expected content to be set in result")
		}
	}
}
