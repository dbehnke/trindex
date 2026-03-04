//go:build integration

package embed

import (
	"testing"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/testutil"
)

func TestClient_Embed_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	embeddingDim := 768
	mockServer := testutil.MockOllamaServer(embeddingDim)
	defer mockServer.Close()

	cfg := &config.Config{
		EmbedBaseURL:    mockServer.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: embeddingDim,
	}

	client := NewClient(cfg)

	embedding, err := client.Embed("test text")
	if err != nil {
		t.Fatalf("Failed to embed text: %v", err)
	}

	if len(embedding) != embeddingDim {
		t.Errorf("Expected embedding dimension %d, got %d", embeddingDim, len(embedding))
	}

	expectedValue := float32(0.0)
	if embedding[0] != expectedValue {
		t.Errorf("Expected first embedding value %f, got %f", expectedValue, embedding[0])
	}
}

func TestClient_EmbedBatch_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	embeddingDim := 768
	mockServer := testutil.MockOllamaServer(embeddingDim)
	defer mockServer.Close()

	cfg := &config.Config{
		EmbedBaseURL:    mockServer.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: embeddingDim,
	}

	client := NewClient(cfg)

	texts := []string{"first text", "second text", "third text"}
	embeddings, err := client.EmbedBatch(texts)
	if err != nil {
		t.Fatalf("Failed to embed batch: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, emb := range embeddings {
		if len(emb) != embeddingDim {
			t.Errorf("Embedding %d: expected dimension %d, got %d", i, embeddingDim, len(emb))
		}
	}
}

func TestClient_ValidateDimensions_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	embeddingDim := 768
	mockServer := testutil.MockOllamaServer(embeddingDim)
	defer mockServer.Close()

	cfg := &config.Config{
		EmbedBaseURL:    mockServer.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: embeddingDim,
	}

	client := NewClient(cfg)

	err := client.ValidateDimensions()
	if err != nil {
		t.Fatalf("Expected dimension validation to pass: %v", err)
	}
}

func TestClient_Embed_WrongDimensions_Integration(t *testing.T) {
	testutil.SkipIfNoDocker(t)

	embeddingDim := 768
	mockServer := testutil.MockOllamaServer(embeddingDim)
	defer mockServer.Close()

	cfg := &config.Config{
		EmbedBaseURL:    mockServer.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 1536,
	}

	client := NewClient(cfg)

	err := client.ValidateDimensions()
	if err == nil {
		t.Error("Expected dimension validation to fail with wrong dimensions")
	}
}
