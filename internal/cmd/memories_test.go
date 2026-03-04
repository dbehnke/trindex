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

func TestMemoriesList(t *testing.T) {
	t.Run("queries api and outputs table", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/memories", r.URL.Path)
			assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))

			memories := []memory.Memory{
				{ID: uuid.New(), Content: "Test memory", Namespace: "default"},
			}
			_ = json.NewEncoder(w).Encode(memories)
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL:     server.URL,
			APIKey:     "test-key",
			Limit:      20,
			JSONOutput: false,
		}

		ctx := context.Background()
		err := runMemoriesListWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("json output", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			memories := []memory.Memory{
				{ID: uuid.New(), Content: "Test memory", Namespace: "default"},
			}
			_ = json.NewEncoder(w).Encode(memories)
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL:     server.URL,
			JSONOutput: true,
		}

		ctx := context.Background()
		err := runMemoriesListWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("namespace filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "myproject", r.URL.Query().Get("namespace"))
			_ = json.NewEncoder(w).Encode([]memory.Memory{})
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL:    server.URL,
			Namespace: "myproject",
		}

		ctx := context.Background()
		err := runMemoriesListWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "database error"}`))
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL: server.URL,
		}

		ctx := context.Background()
		err := runMemoriesListWithFlags(ctx, flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})
}

func TestMemoriesGet(t *testing.T) {
	t.Run("get memory by id", func(t *testing.T) {
		testID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/memories/"+testID.String(), r.URL.Path)
			_ = json.NewEncoder(w).Encode(memory.Memory{
				ID:        testID,
				Content:   "Test content",
				Namespace: "default",
			})
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL: server.URL,
		}

		ctx := context.Background()
		err := runMemoriesGetWithFlags(ctx, testID.String(), flags)

		assert.NoError(t, err)
	})

	t.Run("memory not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL: server.URL,
		}

		ctx := context.Background()
		err := runMemoriesGetWithFlags(ctx, uuid.New().String(), flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid uuid", func(t *testing.T) {
		flags := &MemoriesFlags{}
		ctx := context.Background()
		err := runMemoriesGetWithFlags(ctx, "not-a-uuid", flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}

func TestMemoriesCreate(t *testing.T) {
	t.Run("create memory with content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/memories", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			var req struct {
				Content   string                 `json:"content"`
				Namespace string                 `json:"namespace"`
				Metadata  map[string]interface{} `json:"metadata"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, "Test content", req.Content)
			assert.Equal(t, "default", req.Namespace)

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(memory.Memory{
				ID:        uuid.New(),
				Content:   req.Content,
				Namespace: req.Namespace,
			})
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL:    server.URL,
			Content:   "Test content",
			Namespace: "default",
		}

		ctx := context.Background()
		err := runMemoriesCreateWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("create with metadata", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Content   string                 `json:"content"`
				Namespace string                 `json:"namespace"`
				Metadata  map[string]interface{} `json:"metadata"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)

			assert.Equal(t, "project", req.Metadata["key"])

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(memory.Memory{ID: uuid.New()})
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL:   server.URL,
			Content:  "Test content",
			Metadata: "key=project",
		}

		ctx := context.Background()
		err := runMemoriesCreateWithFlags(ctx, flags)

		assert.NoError(t, err)
	})

	t.Run("missing content", func(t *testing.T) {
		flags := &MemoriesFlags{}
		ctx := context.Background()
		err := runMemoriesCreateWithFlags(ctx, flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content required")
	})
}

func TestMemoriesDelete(t *testing.T) {
	t.Run("delete memory with force", func(t *testing.T) {
		testID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/memories/"+testID.String(), r.URL.Path)
			assert.Equal(t, "DELETE", r.Method)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL: server.URL,
			Force:  true,
		}

		ctx := context.Background()
		err := runMemoriesDeleteWithFlags(ctx, testID.String(), flags)

		assert.NoError(t, err)
	})

	t.Run("memory not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		flags := &MemoriesFlags{
			APIURL: server.URL,
			Force:  true,
		}

		ctx := context.Background()
		err := runMemoriesDeleteWithFlags(ctx, uuid.New().String(), flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid id", func(t *testing.T) {
		flags := &MemoriesFlags{}
		ctx := context.Background()
		err := runMemoriesDeleteWithFlags(ctx, "not-a-uuid", flags)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})
}
