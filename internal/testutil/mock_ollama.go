package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// MockOllamaServer creates a mock embedding server for testing.
// It returns deterministic embeddings based on the requested dimension.
func MockOllamaServer(embeddingDim int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path != "/embeddings" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		data := make([]map[string]interface{}, len(req.Input))
		for i := range req.Input {
			embedding := make([]float32, embeddingDim)
			for j := range embedding {
				embedding[j] = float32(j) * 0.01
			}
			data[i] = map[string]interface{}{
				"object":    "embedding",
				"embedding": embedding,
				"index":     i,
			}
		}

		resp := map[string]interface{}{
			"object": "list",
			"data":   data,
			"model":  req.Model,
			"usage": map[string]int{
				"prompt_tokens": len(req.Input) * 10,
				"total_tokens":  len(req.Input) * 10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// MockOllamaServerWithCustomResponse creates a mock server with a custom handler.
func MockOllamaServerWithCustomResponse(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
