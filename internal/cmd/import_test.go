package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dbehnke/trindex/internal/memory"
	"github.com/stretchr/testify/assert"
)

func TestImportCommand(t *testing.T) {
	t.Run("import from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "import.jsonl")
		_ = os.WriteFile(inputFile, []byte(`{"id":"test-1","content":"memory 1"}`+"\n"), 0644)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/import", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			result := memory.ImportResult{
				Imported: 1,
				Failed:   0,
				Errors:   []string{},
			}
			_ = json.NewEncoder(w).Encode(result)
		}))
		defer server.Close()

		flags := &ImportFlags{
			APIURL: server.URL,
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runImportWithFlags(ctx, inputFile, flags, &buf)

		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Import complete")
		assert.Contains(t, buf.String(), "Imported: 1")
	})

	t.Run("import with skip-existing", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "import.jsonl")
		_ = os.WriteFile(inputFile, []byte(`{"id":"test-1","content":"memory 1"}`+"\n"), 0644)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := memory.ImportResult{
				Imported: 0,
				Failed:   0,
				Errors:   []string{},
			}
			_ = json.NewEncoder(w).Encode(result)
		}))
		defer server.Close()

		flags := &ImportFlags{
			APIURL:       server.URL,
			SkipExisting: "true",
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runImportWithFlags(ctx, inputFile, flags, &buf)

		assert.NoError(t, err)
	})

	t.Run("import with failures", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "import.jsonl")
		_ = os.WriteFile(inputFile, []byte(`{"id":"test-1","content":"memory 1"}`+"\n"), 0644)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := memory.ImportResult{
				Imported: 0,
				Failed:   1,
				Errors:   []string{"invalid format"},
			}
			_ = json.NewEncoder(w).Encode(result)
		}))
		defer server.Close()

		flags := &ImportFlags{
			APIURL: server.URL,
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runImportWithFlags(ctx, inputFile, flags, &buf)

		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Failed: 1")
		assert.Contains(t, buf.String(), "invalid format")
	})

	t.Run("file not found", func(t *testing.T) {
		flags := &ImportFlags{
			APIURL: "http://localhost:8080",
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runImportWithFlags(ctx, "/nonexistent/file.jsonl", flags, &buf)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})

	t.Run("api error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputFile := filepath.Join(tmpDir, "import.jsonl")
		_ = os.WriteFile(inputFile, []byte(`{"content":"test"}`+"\n"), 0644)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "import failed"}`))
		}))
		defer server.Close()

		flags := &ImportFlags{
			APIURL: server.URL,
		}

		var buf bytes.Buffer
		ctx := context.Background()
		err := runImportWithFlags(ctx, inputFile, flags, &buf)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API error")
	})
}
