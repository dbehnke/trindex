package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestServer(t *testing.T) (*Server, func()) {
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
		return nil, func() {}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = database.Migrate(ctx)
	if err != nil {
		database.Close()
		t.Skipf("Migration failed: %v", err)
		return nil, func() {}
	}

	embedClient := embed.NewClient(cfg)
	server := NewServer(cfg, database, embedClient)
	server.RegisterTools()

	cleanup := func() {
		database.Close()
	}

	return server, cleanup
}

func TestServer_RegisterTools(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	if server == nil {
		return
	}

	if server.store == nil {
		t.Error("expected store to be initialized")
	}
	if server.server == nil {
		t.Error("expected mcp server to be initialized")
	}
}

func TestErrorResult(t *testing.T) {
	content := errorResult("TEST_ERROR", "test message")

	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}

	textContent, ok := content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", content[0])
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(textContent.Text), &result); err != nil {
		t.Fatalf("failed to unmarshal error result: %v", err)
	}

	if result["error"] != "TEST_ERROR" {
		t.Errorf("expected error code 'TEST_ERROR', got '%s'", result["error"])
	}
	if result["message"] != "test message" {
		t.Errorf("expected message 'test message', got '%s'", result["message"])
	}
}

func TestSuccessResult(t *testing.T) {
	data := map[string]string{"status": "ok", "id": "123"}
	content := successResult(data)

	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}

	textContent, ok := content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", content[0])
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(textContent.Text), &result); err != nil {
		t.Fatalf("failed to unmarshal success result: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", result["status"])
	}
	if result["id"] != "123" {
		t.Errorf("expected id '123', got '%s'", result["id"])
	}
}

func TestSuccessResult_InvalidData(t *testing.T) {
	content := successResult(make(chan int))

	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}

	textContent, ok := content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", content[0])
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(textContent.Text), &result); err != nil {
		t.Fatalf("failed to unmarshal error result: %v", err)
	}

	if result["error"] != "SERIALIZE_ERROR" {
		t.Errorf("expected error code 'SERIALIZE_ERROR', got '%s'", result["error"])
	}
}

func TestServer_NewServer(t *testing.T) {
	cfg := &config.Config{
		DatabaseURL:     "postgres://localhost/test",
		EmbedBaseURL:    "http://localhost:11434/v1",
		EmbedModel:      "test-model",
		EmbedAPIKey:     "test-key",
		EmbedDimensions: 768,
	}

	database := &db.DB{}
	embedClient := embed.NewClient(cfg)

	server := NewServer(cfg, database, embedClient)

	if server.cfg != cfg {
		t.Error("expected cfg to be set")
	}
	if server.db != database {
		t.Error("expected db to be set")
	}
	if server.embed != embedClient {
		t.Error("expected embed client to be set")
	}
	if server.store == nil {
		t.Error("expected store to be initialized")
	}
	if server.server == nil {
		t.Error("expected mcp server to be initialized")
	}
}
