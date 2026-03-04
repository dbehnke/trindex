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

type StatsFlags struct {
	Namespace  string
	JSONOutput bool
	APIURL     string
	APIKey     string
}

func NewStatsCommand() *Command {
	return &Command{
		Name:        "stats",
		Description: "Show statistics",
		Run: func(ctx context.Context, args []string) error {
			return runStats(ctx, args)
		},
	}
}

func runStats(ctx context.Context, args []string) error {
	flags := &StatsFlags{}
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.StringVar(&flags.Namespace, "namespace", "", "Stats for specific namespace")
	fs.BoolVar(&flags.JSONOutput, "json", false, "Output as JSON")
	fs.StringVar(&flags.APIURL, "api-url", getEnv("TRINDEX_API_URL", "http://localhost:8080"), "API URL")
	fs.StringVar(&flags.APIKey, "api-key", getEnv("TRINDEX_API_KEY", ""), "API key")

	if err := fs.Parse(args); err != nil {
		return err
	}

	return runStatsWithFlags(ctx, flags)
}

func runStatsWithFlags(ctx context.Context, flags *StatsFlags) error {
	url := flags.APIURL + "/api/stats"
	if flags.Namespace != "" {
		url += "?namespace=" + flags.Namespace
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

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

	var stats memory.Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if flags.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(stats)
	}

	fmt.Println("📊 Trindex Statistics")
	fmt.Println()
	fmt.Printf("Total Memories:   %d\n", stats.TotalMemories)
	fmt.Printf("Recent (24h):     %d\n", stats.Recent24h)
	fmt.Printf("Embedding Model:  %s\n", stats.EmbeddingModel)
	fmt.Printf("Dimensions:       %d\n", stats.EmbedDimensions)
	fmt.Println()

	if len(stats.ByNamespace) > 0 {
		fmt.Println("By Namespace:")
		for ns, count := range stats.ByNamespace {
			fmt.Printf("  %s: %d\n", ns, count)
		}
		fmt.Println()
	}

	if len(stats.TopTags) > 0 {
		fmt.Println("Top Tags:")
		for _, tag := range stats.TopTags {
			fmt.Printf("  - %s\n", tag)
		}
	}

	return nil
}
