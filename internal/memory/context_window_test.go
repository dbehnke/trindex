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

func setupTestStoreForContextWindow(t *testing.T) (*Store, func()) {
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

func TestBuildContextWindow(t *testing.T) {
	store, cleanup := setupTestStoreForContextWindow(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := store.Create(ctx, "Architecture decision: Use PostgreSQL with pgvector for vector storage", "project:test", map[string]interface{}{
		"type": "decision",
		"tags": []string{"architecture", "database"},
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	_, err = store.Create(ctx, "User prefers dark mode for all applications", "global", map[string]interface{}{
		"type": "preference",
		"tags": []string{"ui", "user-preference"},
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	_, err = store.Create(ctx, "Bug fixed: Race condition in concurrent access", "project:test", map[string]interface{}{
		"type": "bug",
		"tags": []string{"bug", "concurrency"},
	})
	if err != nil {
		t.Fatalf("Failed to create memory: %v", err)
	}

	params := ContextWindowParams{
		Query:           "database architecture decisions",
		Namespaces:      []string{"project:test", "global"},
		MaxTokens:       2000,
		RelevanceWeight: 0.5,
		RecencyWeight:   0.3,
		TypeBoostWeight: 0.2,
	}

	ranked, err := store.BuildContextWindow(ctx, params)
	if err != nil {
		t.Fatalf("BuildContextWindow failed: %v", err)
	}

	for _, r := range ranked {
		if r.Score <= 0 {
			t.Error("expected positive score")
		}
		if r.Relevance < 0 {
			t.Error("expected non-negative relevance")
		}
		if r.Recency < 0 || r.Recency > 1 {
			t.Errorf("expected recency between 0 and 1, got %f", r.Recency)
		}
		if r.TypeBoost < 0 || r.TypeBoost > 1 {
			t.Errorf("expected type boost between 0 and 1, got %f", r.TypeBoost)
		}
	}
}

func TestBuildContextWindow_RespectsTokenBudget(t *testing.T) {
	store, cleanup := setupTestStoreForContextWindow(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 10; i++ {
		_, err := store.Create(ctx, "This is a test memory with sufficient length to consume tokens when retrieved in a context window", "project:test", nil)
		if err != nil {
			t.Fatalf("Failed to create memory: %v", err)
		}
	}

	params := ContextWindowParams{
		Query:      "test memory",
		Namespaces: []string{"project:test"},
		MaxTokens:  500,
	}

	ranked, err := store.BuildContextWindow(ctx, params)
	if err != nil {
		t.Fatalf("BuildContextWindow failed: %v", err)
	}

	totalTokens := 0
	for _, r := range ranked {
		totalTokens += estimateTokens(r.Content) + 50
	}

	if totalTokens > params.MaxTokens {
		t.Errorf("expected total tokens %d to be within budget %d", totalTokens, params.MaxTokens)
	}
}

func TestTypeBoostScore(t *testing.T) {
	tests := []struct {
		memType  string
		expected float64
	}{
		{"preference", 1.0},
		{"decision", 0.9},
		{"bug", 0.85},
		{"pattern", 0.8},
		{"outcome", 0.7},
		{"fact", 0.6},
		{"unknown", 0.5},
		{"", 0.5},
	}

	for _, tt := range tests {
		got := typeBoostScore(tt.memType)
		if got != tt.expected {
			t.Errorf("typeBoostScore(%q) = %f, want %f", tt.memType, got, tt.expected)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"test", 1},
		{"this is a test", 4},
		{"this is a longer test with more words", 10},
	}

	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.expected {
			t.Errorf("estimateTokens(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestEstimateTokensFromMetadata(t *testing.T) {
	metadata := map[string]interface{}{
		"type": "decision",
		"tags": []interface{}{"architecture", "database"},
	}

	tokens := estimateTokensFromMetadata(metadata)
	if tokens <= 0 {
		t.Error("expected positive token count for non-empty metadata")
	}

	emptyTokens := estimateTokensFromMetadata(nil)
	if emptyTokens != 0 {
		t.Errorf("expected 0 tokens for nil metadata, got %d", emptyTokens)
	}
}

func TestRankMemories(t *testing.T) {
	now := time.Now()
	results := []RecallResult{
		{
			Memory: Memory{
				Content:   "Recent decision",
				CreatedAt: now.Add(-1 * time.Hour),
				Metadata:  map[string]interface{}{"type": "decision"},
			},
			Score: 0.9,
		},
		{
			Memory: Memory{
				Content:   "Old preference",
				CreatedAt: now.Add(-168 * time.Hour),
				Metadata:  map[string]interface{}{"type": "preference"},
			},
			Score: 0.8,
		},
		{
			Memory: Memory{
				Content:   "Old fact",
				CreatedAt: now.Add(-168 * time.Hour),
				Metadata:  map[string]interface{}{"type": "fact"},
			},
			Score: 0.85,
		},
	}

	store := &Store{}
	ranked := store.rankMemories(results)

	if len(ranked) != len(results) {
		t.Errorf("expected %d ranked memories, got %d", len(results), len(ranked))
	}

	if len(ranked) > 0 && ranked[0].TypeBoost != 1.0 {
		t.Logf("Preference type should have highest boost: %f", ranked[0].TypeBoost)
	}

	for _, r := range ranked {
		if r.Score == 0 {
			t.Error("expected non-zero composite score")
		}
	}
}
