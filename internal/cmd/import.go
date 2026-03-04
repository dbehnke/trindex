package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dbehnke/trindex/internal/memory"
)

type ImportFlags struct {
	SkipExisting string
	Namespace    string
	APIURL       string
	APIKey       string
}

func NewImportCommand() *Command {
	return &Command{
		Name:        "import",
		Description: "Import memories",
		Run: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("file required")
			}
			return runImport(ctx, args[0], args[1:])
		},
	}
}

func runImport(ctx context.Context, filePath string, args []string) error {
	flags := &ImportFlags{}
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.StringVar(&flags.SkipExisting, "skip-existing", "false", "Skip duplicates")
	fs.StringVar(&flags.Namespace, "namespace", "", "Import to specific namespace")
	fs.StringVar(&flags.APIURL, "api-url", getEnv("TRINDEX_API_URL", "http://localhost:8080"), "API URL")
	fs.StringVar(&flags.APIKey, "api-key", getEnv("TRINDEX_API_KEY", ""), "API key")

	if err := fs.Parse(args); err != nil {
		return err
	}

	return runImportWithFlags(ctx, filePath, flags, os.Stdout)
}

func runImportWithFlags(ctx context.Context, filePath string, flags *ImportFlags, output io.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	url := flags.APIURL + "/api/import"
	req, err := http.NewRequestWithContext(ctx, "POST", url, file)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if flags.APIKey != "" {
		req.Header.Set("X-API-Key", flags.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result memory.ImportResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	_, _ = fmt.Fprintln(output, "Import complete:")
	_, _ = fmt.Fprintf(output, "  Imported: %d\n", result.Imported)
	if result.Failed > 0 {
		_, _ = fmt.Fprintf(output, "  Failed: %d\n", result.Failed)
		for _, errStr := range result.Errors {
			_, _ = fmt.Fprintf(output, "    - %s\n", errStr)
		}
	}

	return nil
}
