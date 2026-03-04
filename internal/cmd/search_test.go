package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dbehnke/trindex/internal/memory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSearchCommand(t *testing.T) {
	t.Run("search with query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/search", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			var req struct {
				Query string `json:"query"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "machine learning", req.Query)

			result := map[string]interface{}{
				"results": []memory.RecallResult{
					{
						Memory: memory.Memory{ID: uuid.New(), Content: "ML basics", Namespace: "default"},
						Score:  0.95,
					},
				},
				"total": 1,
			}
			_ = json.NewEncoder(w).Encode(result)
		}))
		defer server.Close()

		flags := &SearchFlags{
			APIURL: server.URL,
			TopK:   10,
		}

		ctx := context.Background()
		err := runSearchWithFlags(ctx, "machine learning", flags)

		assert.NoError(t, err)
	})

	t.Run("search with namespace filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Query      string   `json:"query"`
				Namespaces []string `json:"namespaces"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "test query", req.Query)
			assert.Equal(t, []string{"work"}, req.Namespaces)

			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []memory.RecallResult{},
				"total":   0,
			})
		}))
		defer server.Close()

		flags := &SearchFlags{
			APIURL:    server.URL,
			Namespace: "work",
			TopK:      10,
		}

		ctx := context.Background()
		err := runSearchWithFlags(ctx, "test query", flags)

		assert.NoError(t, err)
	})

	t.Run("search with threshold", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Query     string  `json:"query"`
				Threshold float64 `json:"threshold"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, 0.8, req.Threshold)

			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []memory.RecallResult{},
				"total":   0,
			})
		}))
		defer server.Close()

		flags := &SearchFlags{
			APIURL:    server.URL,
			Threshold: 0.8,
		}

		ctx := context.Background()
		err := runSearchWithFlags(ctx, "test", flags)

		assert.NoError(t, err)
	})

	t.Run("no results", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []memory.RecallResult{},
				"total":   0,
			})
		}))
		defer server.Close()

		flags := &SearchFlags{
			APIURL: server.URL,
		}

		ctx := context.Background()
		err := runSearchWithFlags(ctx, "nonexistent", flags)

		assert.NoError(t, err)
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "search failed"}`))
		}))
		defer server.Close()

		flags := &SearchFlags{
			APIURL: server.URL,
		}

		ctx := context.Background()
		err := runSearchWithFlags(ctx, "test", flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})
}
