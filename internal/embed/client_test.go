package embed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dbehnke/trindex/internal/config"
)

func TestClient_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("expected path /embeddings, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type header to be application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header 'Bearer test-key', got '%s'", r.Header.Get("Authorization"))
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}

		if req.Model != "test-model" {
			t.Errorf("expected model 'test-model', got '%s'", req.Model)
		}
		if len(req.Input) != 1 || req.Input[0] != "test text" {
			t.Errorf("expected input ['test text'], got %v", req.Input)
		}

		resp := Response{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
					Index:     0,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		EmbedBaseURL:    server.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 5,
	}

	client := NewClient(cfg)
	embedding, err := client.Embed("test text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embedding) != 5 {
		t.Errorf("expected 5 dimensions, got %d", len(embedding))
	}

	expected := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	for i, v := range embedding {
		if v != expected[i] {
			t.Errorf("expected embedding[%d] = %f, got %f", i, expected[i], v)
		}
	}
}

func TestClient_EmbedBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}

		if len(req.Input) != 2 {
			t.Errorf("expected 2 inputs, got %d", len(req.Input))
		}

		resp := Response{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
				{
					Object:    "embedding",
					Embedding: []float32{0.4, 0.5, 0.6},
					Index:     1,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		EmbedBaseURL:    server.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 3,
	}

	client := NewClient(cfg)
	embeddings, err := client.EmbedBatch([]string{"text 1", "text 2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(embeddings) != 2 {
		t.Errorf("expected 2 embeddings, got %d", len(embeddings))
	}
}

func TestClient_Embed_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		EmbedBaseURL:    server.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 5,
	}

	client := NewClient(cfg)
	_, err := client.Embed("test text")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestClient_Embed_NetworkError(t *testing.T) {
	cfg := &config.Config{
		EmbedBaseURL:    "http://localhost:99999",
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 5,
	}

	client := NewClient(cfg)
	_, err := client.Embed("test text")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestClient_EmptyInput(t *testing.T) {
	cfg := &config.Config{
		EmbedBaseURL:    "http://localhost",
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 5,
	}

	client := NewClient(cfg)
	_, err := client.EmbedBatch([]string{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestClient_ValidateDimensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
					Index:     0,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		EmbedBaseURL:    server.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 5,
	}

	client := NewClient(cfg)
	err := client.ValidateDimensions()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClient_ValidateDimensions_Mismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Object: "list",
			Data: []struct {
				Object    string    `json:"object"`
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{
					Object:    "embedding",
					Embedding: []float32{0.1, 0.2, 0.3},
					Index:     0,
				},
			},
			Model: "test-model",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		EmbedBaseURL:    server.URL,
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 10,
	}

	client := NewClient(cfg)
	err := client.ValidateDimensions()
	if err == nil {
		t.Error("expected error for dimension mismatch")
	}
}

func TestClient_Model(t *testing.T) {
	cfg := &config.Config{
		EmbedBaseURL:    "http://localhost",
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 5,
	}

	client := NewClient(cfg)
	if client.Model() != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", client.Model())
	}
}

func TestClient_Dimensions(t *testing.T) {
	cfg := &config.Config{
		EmbedBaseURL:    "http://localhost",
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 768,
	}

	client := NewClient(cfg)
	if client.Dimensions() != 768 {
		t.Errorf("expected dimensions 768, got %d", client.Dimensions())
	}
}
