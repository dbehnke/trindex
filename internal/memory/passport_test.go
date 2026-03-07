package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/testutil"
)

func setupTestStoreForPassport(t *testing.T) (*Store, func()) {
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

func TestCreatePassport(t *testing.T) {
	store, cleanup := setupTestStoreForPassport(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Architecture decision for Linear integration", "project:linear", map[string]interface{}{
		"type": "decision",
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	_, err = store.Create(ctx, "User prefers to track issues in Linear", "global", map[string]interface{}{
		"type": "preference",
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	params := PassportParams{
		Source:     "trindex",
		Target:     "linear",
		Query:      "Linear integration preferences",
		Namespaces: []string{"project:linear", "global"},
		TopK:       5,
		TTLSeconds: 3600,
	}

	passport, err := store.CreatePassport(ctx, params)
	if err != nil {
		t.Fatalf("CreatePassport failed: %v", err)
	}

	if passport.ID.String() == "" {
		t.Error("expected passport ID to be set")
	}

	if passport.Source != "trindex" {
		t.Errorf("expected source 'trindex', got '%s'", passport.Source)
	}

	if passport.Target != "linear" {
		t.Errorf("expected target 'linear', got '%s'", passport.Target)
	}

	if passport.TTLSeconds != 3600 {
		t.Errorf("expected TTL 3600, got %d", passport.TTLSeconds)
	}

	if passport.ExpiresAt.Before(time.Now()) {
		t.Error("expected expiry in the future")
	}
}

func TestCreatePassport_Defaults(t *testing.T) {
	store, cleanup := setupTestStoreForPassport(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Some context to transfer", "test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	params := PassportParams{
		Source:     "system-a",
		Target:     "system-b",
		Query:      "context",
		Namespaces: []string{"test"},
	}

	passport, err := store.CreatePassport(ctx, params)
	if err != nil {
		t.Fatalf("CreatePassport failed: %v", err)
	}

	if passport.TopK != 10 {
		t.Errorf("expected default TopK 10, got %d", passport.TopK)
	}

	if passport.TTLSeconds != 3600 {
		t.Errorf("expected default TTL 3600, got %d", passport.TTLSeconds)
	}
}

func TestImportPassport(t *testing.T) {
	store, cleanup := setupTestStoreForPassport(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mem, err := store.Create(ctx, "Context to import", "source-ns", map[string]interface{}{
		"type": "decision",
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	passport := &ContextPassport{
		ID:        mem.ID,
		Source:    "source-system",
		Target:    "target-system",
		Query:     "test query",
		Memories:  []Memory{*mem},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	imported, err := store.ImportPassport(ctx, passport, "target-ns")
	if err != nil {
		t.Fatalf("ImportPassport failed: %v", err)
	}

	if imported != 1 {
		t.Errorf("expected 1 imported memory, got %d", imported)
	}

	list, err := store.List(ctx, ListParams{Namespace: "target-ns"})
	if err != nil {
		t.Fatalf("Failed to list memories: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("expected 1 memory in target-ns, got %d", len(list))
	}

	if len(list) > 0 {
		if list[0].Metadata["imported_from"] != "source-system" {
			t.Error("expected imported_from metadata to be set")
		}
		if list[0].Metadata["passport_id"] != passport.ID.String() {
			t.Error("expected passport_id metadata to be set")
		}
	}
}

func TestImportPassport_SkipsExpired(t *testing.T) {
	store, cleanup := setupTestStoreForPassport(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	expiredTime := time.Now().Add(-1 * time.Hour)
	mem := &Memory{
		Content:   "Expired memory",
		Namespace: "test",
		Metadata:  map[string]interface{}{},
		ExpiresAt: &expiredTime,
	}

	passport := &ContextPassport{
		Source:    "source",
		Target:    "target",
		Memories:  []Memory{*mem},
		CreatedAt: time.Now(),
	}

	imported, err := store.ImportPassport(ctx, passport, "target-ns")
	if err != nil {
		t.Fatalf("ImportPassport failed: %v", err)
	}

	if imported != 0 {
		t.Errorf("expected 0 imported memories (expired), got %d", imported)
	}
}

func TestImportPassport_DefaultNamespace(t *testing.T) {
	store, cleanup := setupTestStoreForPassport(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mem, err := store.Create(ctx, "Test memory", "test", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	passport := &ContextPassport{
		Source:    "source",
		Target:    "target",
		Memories:  []Memory{*mem},
		CreatedAt: time.Now(),
	}

	imported, err := store.ImportPassport(ctx, passport, "")
	if err != nil {
		t.Fatalf("ImportPassport failed: %v", err)
	}

	if imported != 1 {
		t.Errorf("expected 1 imported memory, got %d", imported)
	}

	list, err := store.List(ctx, ListParams{Namespace: "default"})
	if err != nil {
		t.Fatalf("Failed to list memories: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("expected 1 memory in default namespace, got %d", len(list))
	}
}

func TestContextPassport_Expiry(t *testing.T) {
	now := time.Now()
	passport := &ContextPassport{
		ID:         [16]byte{1, 2, 3, 4},
		Source:     "test-source",
		Target:     "test-target",
		Query:      "test",
		Memories:   []Memory{},
		CreatedAt:  now,
		ExpiresAt:  now.Add(1 * time.Hour),
		TTLSeconds: 3600,
	}

	if passport.ExpiresAt.Before(now) {
		t.Error("passport expiry should be in the future")
	}

	if passport.TTLSeconds != 3600 {
		t.Errorf("expected TTL 3600, got %d", passport.TTLSeconds)
	}
}
