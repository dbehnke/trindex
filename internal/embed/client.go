package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dbehnke/trindex/internal/config"
)

// Client is an OpenAI-compatible embedding client
type Client struct {
	cfg    *config.Config
	client *http.Client
}

// Request represents an OpenAI embedding request
type Request struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// Response represents an OpenAI embedding response
type Response struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient creates a new embedding client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed generates embeddings for a single text
func (c *Client) Embed(text string) ([]float32, error) {
	embeddings, err := c.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (c *Client) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	req := Request{
		Model: c.cfg.EmbedModel,
		Input: texts,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.cfg.EmbedBaseURL + "/embeddings"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.EmbedAPIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding request failed with status %d", resp.StatusCode)
	}

	var embedResp Response
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make([][]float32, len(embedResp.Data))
	for i, d := range embedResp.Data {
		result[i] = d.Embedding
	}

	return result, nil
}

// ValidateDimensions checks if the embedding endpoint returns the expected dimensions
func (c *Client) ValidateDimensions() error {
	testText := "test"
	embedding, err := c.Embed(testText)
	if err != nil {
		return fmt.Errorf("failed to generate test embedding: %w", err)
	}

	if len(embedding) != c.cfg.EmbedDimensions {
		return fmt.Errorf("dimension mismatch: expected %d, got %d. Update EMBED_DIMENSIONS env var",
			c.cfg.EmbedDimensions, len(embedding))
	}

	return nil
}

// Model returns the configured embedding model
func (c *Client) Model() string {
	return c.cfg.EmbedModel
}

// Dimensions returns the configured embedding dimensions
func (c *Client) Dimensions() int {
	return c.cfg.EmbedDimensions
}
