package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dbehnke/trindex/internal/memory"
)

type SearchFlags struct {
	Namespace  string
	TopK       int
	Threshold  float64
	JSONOutput bool
	APIURL     string
	APIKey     string
}

func NewSearchCommand() *Command {
	return &Command{
		Name:        "search",
		Description: "Search memories",
		Run: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("query required")
			}
			query := strings.Join(args, " ")
			return runSearch(ctx, query)
		},
	}
}

func runSearch(ctx context.Context, query string) error {
	flags := &SearchFlags{}
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.StringVar(&flags.Namespace, "namespace", "", "Search namespace (repeatable)")
	fs.IntVar(&flags.TopK, "top-k", 10, "Number of results")
	fs.Float64Var(&flags.Threshold, "threshold", 0.0, "Similarity threshold (0.0-1.0)")
	fs.BoolVar(&flags.JSONOutput, "json", false, "Output as JSON")
	fs.StringVar(&flags.APIURL, "api-url", getEnv("TRINDEX_API_URL", "http://localhost:9636"), "API URL")
	fs.StringVar(&flags.APIKey, "api-key", getEnv("TRINDEX_API_KEY", ""), "API key")

	if err := fs.Parse(os.Args[3:]); err != nil {
		return err
	}

	return runSearchWithFlags(ctx, query, flags)
}

func runSearchWithFlags(ctx context.Context, query string, flags *SearchFlags) error {
	var namespaces []string
	if flags.Namespace != "" {
		namespaces = []string{flags.Namespace}
	}

	reqBody := map[string]interface{}{
		"query":      query,
		"namespaces": namespaces,
		"top_k":      flags.TopK,
		"threshold":  flags.Threshold,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := flags.APIURL + "/api/search"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
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

	var result struct {
		Results []memory.RecallResult `json:"results"`
		Total   int                   `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if flags.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if len(result.Results) == 0 {
		fmt.Println("No results found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SCORE\tID\tNAMESPACE\tCONTENT")
	for _, r := range result.Results {
		content := r.Content
		if len(content) > 50 {
			content = content[:47] + "..."
		}
		_, _ = fmt.Fprintf(w, "%.3f\t%s\t%s\t%s\n",
			r.Score, r.ID.String(), r.Namespace, content)
	}
	return w.Flush()
}
