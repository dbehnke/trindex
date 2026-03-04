package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExportCommand(t *testing.T) {
	t.Run("export to stdout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/export", r.URL.Path)
			_, _ = w.Write([]byte(`{"id":"test-1","content":"memory 1"}` + "\n"))
			_, _ = w.Write([]byte(`{"id":"test-2","content":"memory 2"}` + "\n"))
		}))
		defer server.Close()

		flags := &ExportFlags{
			APIURL: server.URL,
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runExportWithFlags(ctx, flags, &buf)

		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "test-1")
		assert.Contains(t, buf.String(), "Export complete")
	})

	t.Run("export with namespace filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "work", r.URL.Query().Get("namespace"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		flags := &ExportFlags{
			APIURL:    server.URL,
			Namespace: "work",
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runExportWithFlags(ctx, flags, &buf)

		assert.NoError(t, err)
	})

	t.Run("export with date range", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "2024-01-01T00:00:00Z", r.URL.Query().Get("since"))
			assert.Equal(t, "2024-12-31T23:59:59Z", r.URL.Query().Get("until"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		flags := &ExportFlags{
			APIURL: server.URL,
			Since:  "2024-01-01T00:00:00Z",
			Until:  "2024-12-31T23:59:59Z",
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runExportWithFlags(ctx, flags, &buf)

		assert.NoError(t, err)
	})

	t.Run("invalid since date", func(t *testing.T) {
		flags := &ExportFlags{
			APIURL: "http://localhost:8080",
			Since:  "invalid-date",
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runExportWithFlags(ctx, flags, &buf)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid since date")
	})

	t.Run("invalid until date", func(t *testing.T) {
		flags := &ExportFlags{
			APIURL: "http://localhost:8080",
			Until:  "invalid-date",
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runExportWithFlags(ctx, flags, &buf)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid until date")
	})

	t.Run("api error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "export failed"}`))
		}))
		defer server.Close()

		flags := &ExportFlags{
			APIURL: server.URL,
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runExportWithFlags(ctx, flags, &buf)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})

	t.Run("export to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "export.jsonl")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"id":"test-1","content":"memory 1"}` + "\n"))
		}))
		defer server.Close()

		flags := &ExportFlags{
			APIURL: server.URL,
			Output: outputFile,
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runExportWithFlags(ctx, flags, &buf)

		assert.NoError(t, err)

		data, err := os.ReadFile(outputFile)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "test-1")
	})
}
