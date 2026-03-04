package embed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dbehnke/trindex/internal/config"
)

type Client struct {
	cfg    *config.Config
	client *http.Client
}

type Request struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

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

func NewClient(cfg *config.Config) *Client {
	timeout := time.Duration(cfg.EmbedRequestTimeout) * time.Second
	if timeout == 0 {
	timeout = 30 * time.Second
	}

	return &Client{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

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

func (c *Client) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	maxRetries := c.cfg.EmbedMaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	retryDelay := time.Duration(c.cfg.EmbedRetryDelay) * time.Millisecond
	if retryDelay == 0 {
		retryDelay = 1000 * time.Millisecond
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay * time.Duration(attempt))
		}

		result, err := c.doEmbedRequest(texts)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (c *Client) doEmbedRequest(texts []string) ([][]float32, error) {
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

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary",
		"too many requests",
		"service unavailable",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

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

func (c *Client) Model() string {
	return c.cfg.EmbedModel
}

func (c *Client) Dimensions() int {
	return c.cfg.EmbedDimensions
}
