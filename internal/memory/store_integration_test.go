//go:build integration

package memory

import (
	"context"
	"testing"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/testutil"
)

func setupTestDBForIntegration(ctx context.Context, connStr string, embeddingDims int) (*db.DB, error) {
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
		return nil, err
	}

	if err := database.Migrate(ctx); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func TestStore_Create_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := setupTestDBForIntegration(ctx, container.ConnStr, embeddingDim)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	mockServer := testutil.MockOllamaServer(embeddingDim)
	defer mockServer.Close()

	cfg := &config.Config{
		EmbedBaseURL:               mockServer.URL,
		EmbedModel:                 "test-model",
		EmbedAPIKey:                "test-key",
		EmbedDimensions:            embeddingDim,
		DefaultTopK:                10,
		DefaultSimilarityThreshold: 0.7,
		HybridVectorWeight:         0.7,
		HybridFTSWeight:            0.3,
	}

	embedClient := embed.NewClient(cfg)
	store := NewStore(db, embedClient, cfg)

	memory, err := store.Create(ctx, "Test memory content", "test-namespace", map[string]interface{}{
		"tag": "test",
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	if memory.ID.String() == "" {
		t.Error("Expected memory ID to be set")
	}

	if memory.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", memory.Namespace)
	}

	if memory.Content != "Test memory content" {
		t.Errorf("Expected content 'Test memory content', got '%s'", memory.Content)
	}

	if memory.Metadata["tag"] != "test" {
		t.Error("Expected metadata to contain tag")
	}
}

func TestStore_Recall_HybridSearch_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := setupTestDBForIntegration(ctx, container.ConnStr, embeddingDim)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	mockServer := testutil.MockOllamaServer(embeddingDim)
	defer mockServer.Close()

	cfg := &config.Config{
		EmbedBaseURL:               mockServer.URL,
		EmbedModel:                 "test-model",
		EmbedAPIKey:                "test-key",
		EmbedDimensions:            embeddingDim,
		DefaultTopK:                10,
		DefaultSimilarityThreshold: 0.01,
		HybridVectorWeight:         0.7,
		HybridFTSWeight:            0.3,
	}

	embedClient := embed.NewClient(cfg)
	store := NewStore(db, embedClient, cfg)

	_, err = store.Create(ctx, "Database architecture decisions for Postgres", "project-a", nil)
	if err != nil {
		t.Fatalf("Failed to create memory 1: %v", err)
	}

	_, err = store.Create(ctx, "API design patterns for REST endpoints", "project-b", nil)
	if err != nil {
		t.Fatalf("Failed to create memory 2: %v", err)
	}

	_, err = store.Create(ctx, "PostgreSQL indexing strategies with pgvector", "global", nil)
	if err != nil {
		t.Fatalf("Failed to create memory 3: %v", err)
	}

	results, err := store.Recall(ctx, RecallParams{
		Query:      "database postgres",
		Namespaces: []string{"project-a"},
		TopK:       5,
		Threshold:  0.01,
	})
	if err != nil {
		t.Fatalf("Failed to recall: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected some results from recall")
	}

	foundGlobal := false
	for _, r := range results {
		if r.Namespace == "global" {
			foundGlobal = true
			break
		}
	}

	if !foundGlobal {
		t.Error("Expected global namespace to be included in results")
	}
}

func TestStore_Recall_UserDave_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := setupTestDBForIntegration(ctx, container.ConnStr, embeddingDim)
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	defer db.Close()

	mockServer := testutil.MockOllamaServer(embeddingDim)
	defer mockServer.Close()

	cfg := &config.Config{
		EmbedBaseURL:               mockServer.URL,
		EmbedModel:                 "test-model",
		EmbedAPIKey:                "test-key",
		EmbedDimensions:            embeddingDim,
		DefaultTopK:                5,
		DefaultSimilarityThreshold: 0.0001,
		HybridVectorWeight:         0.7,
		HybridFTSWeight:            0.3,
		DefaultNamespace:           "default",
	}

	embedClient := embed.NewClient(cfg)
	store := NewStore(db, embedClient, cfg)

	// OpenCode Action 1: Create Memory
	_, err = store.Create(ctx, "The user's name is Dave", "default", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	// OpenCode Action 2: Recall Memory
	results, err := store.Recall(ctx, RecallParams{
		Query:      "user name identity",
		Namespaces: []string{}, // OpenCode omitted namespaces
		TopK:       5,
		Threshold:  0.0001,
	})
	if err != nil {
		t.Fatalf("Failed to recall: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Recall returned 0 results for 'user name identity', but memory exists.")
	}

	foundDave := false
	for _, r := range results {
		if r.Content == "The user's name is Dave" {
			foundDave = true
			t.Logf("Successfully found Dave with score: %f", r.Score)
			break
		}
	}

	if !foundDave {
		t.Error("Expected memory containing Dave to be included in results")
	}
}
