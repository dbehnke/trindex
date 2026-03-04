package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dbehnke/trindex/internal/memory"
	"github.com/stretchr/testify/assert"
)

func TestStatsCommand(t *testing.T) {
	t.Run("get stats", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/stats", r.URL.Path)
			assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))

			stats := memory.Stats{
				TotalMemories:   100,
				Recent24h:       5,
				EmbeddingModel:  "nomic-embed-text",
				EmbedDimensions: 768,
				ByNamespace:     map[string]int64{"default": 80, "work": 20},
				TopTags:         []string{"ai", "ml", "go"},
			}
			_ = json.NewEncoder(w).Encode(stats)
		}))
		defer server.Close()

		flags := &StatsFlags{
			APIURL: server.URL,
			APIKey: "test-key",
		}

		ctx := context.Background()
		err := runStatsWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("stats with namespace filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "work", r.URL.Query().Get("namespace"))
			stats := memory.Stats{
				TotalMemories: 20,
				ByNamespace:   map[string]int64{"work": 20},
			}
			_ = json.NewEncoder(w).Encode(stats)
		}))
		defer server.Close()

		flags := &StatsFlags{
			APIURL:    server.URL,
			Namespace: "work",
		}

		ctx := context.Background()
		err := runStatsWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("json output", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stats := memory.Stats{
				TotalMemories: 50,
			}
			_ = json.NewEncoder(w).Encode(stats)
		}))
		defer server.Close()

		flags := &StatsFlags{
			APIURL:     server.URL,
			JSONOutput: true,
		}

		ctx := context.Background()
		err := runStatsWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "database error"}`))
		}))
		defer server.Close()

		flags := &StatsFlags{
			APIURL: server.URL,
		}

		ctx := context.Background()
		err := runStatsWithFlags(ctx, flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})
}
