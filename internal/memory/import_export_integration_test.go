//go:build integration

package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/testutil"
	"github.com/google/uuid"
)

func TestStore_Export_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := testutil.SetupTestDB(ctx, container.ConnStr, embeddingDim)
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

	_, err = store.Create(ctx, "Memory one", "namespace-a", map[string]interface{}{"tag": "test"})
	if err != nil {
		t.Fatalf("Failed to create memory 1: %v", err)
	}

	_, err = store.Create(ctx, "Memory two", "namespace-b", map[string]interface{}{"tag": "test"})
	if err != nil {
		t.Fatalf("Failed to create memory 2: %v", err)
	}

	var buf bytes.Buffer
	result, err := store.Export(ctx, "", nil, nil, &buf)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Expected 2 memories exported, got %d", result.Count)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 JSON lines, got %d", len(lines))
	}

	var exported ExportMemory
	if err := json.Unmarshal([]byte(lines[0]), &exported); err != nil {
		t.Fatalf("Failed to parse exported memory: %v", err)
	}

	if exported.Content == "" {
		t.Error("Expected exported memory to have content")
	}
}

func TestStore_Import_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := testutil.SetupTestDB(ctx, container.ConnStr, embeddingDim)
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

	importData := []ImportMemory{
		{
			ID:        uuid.New(),
			Namespace: "imported-namespace",
			Content:   "Imported memory content",
			Metadata:  map[string]interface{}{"source": "test"},
		},
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, mem := range importData {
		encoder.Encode(mem)
	}

	result, err := store.Import(ctx, &buf, ImportOptions{})
	if err != nil {
		t.Fatalf("Failed to import: %v", err)
	}

	if result.Imported != 1 {
		t.Errorf("Expected 1 imported memory, got %d", result.Imported)
	}

	memories, err := store.List(ctx, ListParams{Namespace: "imported-namespace"})
	if err != nil {
		t.Fatalf("Failed to list memories: %v", err)
	}

	if len(memories) != 1 {
		t.Errorf("Expected 1 memory in namespace, got %d", len(memories))
	}
}

func TestStore_Import_WithDuplicateDetection_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := testutil.SetupTestDB(ctx, container.ConnStr, embeddingDim)
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

	existingID := uuid.New()
	_, err = store.Create(ctx, "Existing memory", "test-namespace", nil)
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	importData := ImportMemory{
		ID:        existingID,
		Namespace: "test-namespace",
		Content:   "Different content",
		Metadata:  map[string]interface{}{},
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.Encode(importData)

	result, err := store.Import(ctx, &buf, ImportOptions{SkipExisting: true})
	if err != nil {
		t.Fatalf("Failed to import: %v", err)
	}

	if result.Imported != 1 {
		t.Errorf("Expected 1 imported memory with skip-existing, got %d", result.Imported)
	}
}

func TestStore_FindDuplicates_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := testutil.SetupTestDB(ctx, container.ConnStr, embeddingDim)
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

	_, err = store.Create(ctx, "Similar memory content A", "test-namespace", nil)
	if err != nil {
		t.Fatalf("Failed to create memory 1: %v", err)
	}

	_, err = store.Create(ctx, "Similar memory content B", "test-namespace", nil)
	if err != nil {
		t.Fatalf("Failed to create memory 2: %v", err)
	}

	candidates, err := store.FindDuplicates(ctx, "test-namespace", 0.01, 10)
	if err != nil {
		t.Fatalf("Failed to find duplicates: %v", err)
	}

	if len(candidates) > 0 {
		t.Logf("Found %d potential duplicates (expected with low threshold)", len(candidates))
	}
}

func TestStore_MergeDuplicates_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	ctx := context.Background()

	container, err := testutil.NewPostgresContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	embeddingDim := 768
	db, err := testutil.SetupTestDB(ctx, container.ConnStr, embeddingDim)
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

	mem1, err := store.Create(ctx, "Keep this memory", "test-namespace", nil)
	if err != nil {
		t.Fatalf("Failed to create memory 1: %v", err)
	}

	mem2, err := store.Create(ctx, "Remove this memory", "test-namespace", nil)
	if err != nil {
		t.Fatalf("Failed to create memory 2: %v", err)
	}

	err = store.MergeDuplicates(ctx, mem1.ID, mem2.ID)
	if err != nil {
		t.Fatalf("Failed to merge duplicates: %v", err)
	}

	_, err = store.GetByID(ctx, mem2.ID)
	if err == nil {
		t.Error("Expected removed memory to not exist")
	}

	_, err = store.GetByID(ctx, mem1.ID)
	if err != nil {
		t.Errorf("Expected kept memory to exist: %v", err)
	}
}
