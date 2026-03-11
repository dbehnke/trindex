package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/testutil"
	"github.com/google/uuid"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	embeddingDim := 768
	cfg := &config.Config{
		DatabaseURL:        container.ConnStr,
		EmbedDimensions:    embeddingDim,
		HNSWM:              16,
		HNSWEfConstruction: 64,
		HNSWEfSearch:       40,
		DBMaxConns:         10,
		DBMinConns:         2,
		DBMaxConnLifetime:  60,
		DBMaxConnIdleTime:  30,
		DefaultNamespace:   "default",
	}

	database, err := db.New(cfg)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to create database: %v", err)
	}

	if err := database.Migrate(ctx); err != nil {
		database.Close()
		_ = container.Terminate(ctx)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	mockServer := testutil.MockOllamaServer(embeddingDim)

	embedCfg := &config.Config{
		EmbedBaseURL:               mockServer.URL,
		EmbedModel:                 "test-model",
		EmbedAPIKey:                "test-key",
		EmbedDimensions:            embeddingDim,
		DefaultTopK:                10,
		DefaultSimilarityThreshold: 0.7,
		HNSWM:                      16,
		HNSWEfConstruction:         64,
		HNSWEfSearch:               40,
		DefaultNamespace:           "default",
		HybridVectorWeight:         0.7,
		HybridFTSWeight:            0.3,
	}

	embedClient := embed.NewClient(embedCfg)
	store := NewStore(database, embedClient, embedCfg)

	cleanup := func() {
		mockServer.Close()
		database.Close()
		_ = container.Terminate(ctx)
	}

	return store, cleanup
}

func TestStore_Create(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	memory, err := store.Create(ctx, "Test memory content", "test-namespace", map[string]interface{}{
		"tags": []string{"test", "memory"},
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
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
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	memory, err := store.Create(ctx, "Test content", "", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	if memory.Namespace != "default" {
		t.Errorf("expected default namespace, got '%s'", memory.Namespace)
	}
}

func TestStore_GetByID(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := store.Create(ctx, "Test memory for get", "test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
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
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.GetByID(ctx, uuid.New())
	if err == nil {
		t.Error("expected error for non-existent memory")
	}
}

func TestStore_List(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Memory 1", "list-test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}
	_, err = store.Create(ctx, "Memory 2", "list-test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

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
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created, err := store.Create(ctx, "Memory to delete", "test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
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
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Memory 1", "delete-ns", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}
	_, err = store.Create(ctx, "Memory 2", "delete-ns", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

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
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Stats test memory", "stats-test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
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
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "The quick brown fox jumps over the lazy dog", "recall-test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	params := RecallParams{
		Query:      "quick fox",
		Namespaces: []string{"recall-test"},
		TopK:       5,
		Threshold:  0.01,
		Filter:     Filter{},
	}

	results, err := store.Recall(ctx, params)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
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

func TestStore_CreateWithParams_Deduplication(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	content := "This is a unique memory for dedup testing"
	namespace := "dedup-test"

	mem1, err := store.CreateWithParams(ctx, CreateParams{
		Content:   content,
		Namespace: namespace,
		Metadata:  map[string]interface{}{"test": "value"},
	})
	if err != nil {
		t.Fatalf("Failed to create first memory: %v", err)
	}

	mem2, err := store.CreateWithParams(ctx, CreateParams{
		Content:   content,
		Namespace: namespace,
		Metadata:  map[string]interface{}{"test": "different"},
	})
	if err != nil {
		t.Fatalf("Failed to create second memory: %v", err)
	}

	if mem1.ID != mem2.ID {
		t.Error("expected duplicate content to return existing memory")
	}

	if mem2.ContentHash == "" {
		t.Error("expected content_hash to be set")
	}
}

func TestStore_CreateWithParams_TTL(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mem, err := store.CreateWithParams(ctx, CreateParams{
		Content:    "Memory with TTL",
		Namespace:  "ttl-test",
		TTLSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("Failed to create memory with TTL: %v", err)
	}

	if mem.TTLSeconds != 3600 {
		t.Errorf("expected TTLSeconds to be 3600, got %d", mem.TTLSeconds)
	}

	if mem.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}

	expectedExpiry := time.Now().Add(3600 * time.Second)
	if mem.ExpiresAt.Before(expectedExpiry.Add(-5*time.Second)) || mem.ExpiresAt.After(expectedExpiry.Add(5*time.Second)) {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, mem.ExpiresAt)
	}
}

func TestStore_CreateWithParams_NoTTL(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mem, err := store.CreateWithParams(ctx, CreateParams{
		Content:   "Memory without TTL",
		Namespace: "no-ttl-test",
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	if mem.ExpiresAt != nil {
		t.Error("expected ExpiresAt to be nil when no TTL specified")
	}

	if mem.TTLSeconds != 0 {
		t.Errorf("expected TTLSeconds to be 0, got %d", mem.TTLSeconds)
	}
}

func TestStore_DeleteExpired(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mem, err := store.CreateWithParams(ctx, CreateParams{
		Content:    "Memory that will expire",
		Namespace:  "expire-test",
		TTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	_, err = store.GetByID(ctx, mem.ID)
	if err != nil {
		t.Fatalf("Should be able to retrieve memory before expiry: %v", err)
	}

	time.Sleep(2 * time.Second)

	deleted, err := store.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("Failed to delete expired memories: %v", err)
	}

	if deleted < 1 {
		t.Errorf("expected at least 1 expired memory deleted, got %d", deleted)
	}

	_, err = store.GetByID(ctx, mem.ID)
	if err == nil {
		t.Error("expected memory to be deleted after expiry")
	}
}

func TestStore_Recall_FiltersExpired(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.CreateWithParams(ctx, CreateParams{
		Content:    "Unique content for expired recall test",
		Namespace:  "expired-recall-test",
		TTLSeconds: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	time.Sleep(2 * time.Second)

	params := RecallParams{
		Query:      "unique content expired recall",
		Namespaces: []string{"expired-recall-test"},
		TopK:       5,
		Threshold:  0.01,
	}

	results, err := store.Recall(ctx, params)
	if err != nil {
		t.Fatalf("Recall failed: %v", err)
	}

	for _, r := range results {
		if r.Namespace == "expired-recall-test" {
			t.Error("expected expired memories to be filtered from recall results")
		}
	}
}

func TestComputeContentHash(t *testing.T) {
	hash1 := computeContentHash("test content")
	hash2 := computeContentHash("test content")
	hash3 := computeContentHash("different content")

	if hash1 != hash2 {
		t.Error("same content should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("different content should produce different hash")
	}

	if len(hash1) != 64 {
		t.Errorf("expected SHA256 hash length of 64, got %d", len(hash1))
	}
}
