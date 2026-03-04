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
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/dbehnke/trindex/internal/memory"
	"github.com/google/uuid"
)

type MemoriesFlags struct {
	Namespace  string
	Limit      int
	Offset     int
	Order      string
	JSONOutput bool
	APIURL     string
	APIKey     string
	Content    string
	Metadata   string
	Force      bool
	File       string
}

func NewMemoriesCommand() *Command {
	return &Command{
		Name:        "memories",
		Description: "Memory operations (list, get, create, delete)",
		Run: func(ctx context.Context, args []string) error {
			if len(args) < 1 {
				printMemoriesHelp()
				return fmt.Errorf("subcommand required")
			}

			subcommand := args[0]
			switch subcommand {
			case "list":
				return runMemoriesListCLI(ctx, args[1:])
			case "get":
				return runMemoriesGetCLI(ctx, args[1:])
			case "create":
				return runMemoriesCreateCLI(ctx, args[1:])
			case "delete":
				return runMemoriesDeleteCLI(ctx, args[1:])
			default:
				return fmt.Errorf("unknown subcommand: %s", subcommand)
			}
		},
	}
}

func printMemoriesHelp() {
	fmt.Println("Usage: trindex memories <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list    List memories")
	fmt.Println("  get     Get a memory by ID")
	fmt.Println("  create  Create a new memory")
	fmt.Println("  delete  Delete a memory")
	fmt.Println()
	fmt.Println("Run 'trindex memories <subcommand> --help' for more information.")
}

func parseMemoriesFlags(args []string) (*MemoriesFlags, error) {
	flags := &MemoriesFlags{}
	fs := flag.NewFlagSet("memories", flag.ContinueOnError)
	fs.StringVar(&flags.Namespace, "namespace", "", "Filter by namespace")
	fs.IntVar(&flags.Limit, "limit", 20, "Limit results")
	fs.IntVar(&flags.Offset, "offset", 0, "Pagination offset")
	fs.StringVar(&flags.Order, "order", "desc", "Sort order (asc|desc)")
	fs.BoolVar(&flags.JSONOutput, "json", false, "Output as JSON")
	fs.StringVar(&flags.APIURL, "api-url", getEnv("TRINDEX_API_URL", "http://localhost:8080"), "API URL")
	fs.StringVar(&flags.APIKey, "api-key", getEnv("TRINDEX_API_KEY", ""), "API key")
	fs.StringVar(&flags.Content, "content", "", "Memory content")
	fs.StringVar(&flags.Metadata, "metadata", "", "Metadata key=value (repeatable)")
	fs.BoolVar(&flags.Force, "force", false, "Skip confirmation")
	fs.StringVar(&flags.File, "file", "", "Read content from file")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return flags, nil
}

func runMemoriesListCLI(ctx context.Context, args []string) error {
	flags, err := parseMemoriesFlags(args)
	if err != nil {
		return err
	}
	return runMemoriesListWithFlags(ctx, flags)
}

func runMemoriesListWithFlags(ctx context.Context, flags *MemoriesFlags) error {
	params := "?"
	if flags.Namespace != "" {
		params += "namespace=" + flags.Namespace + "&"
	}
	params += "limit=" + strconv.Itoa(flags.Limit) + "&"
	params += "offset=" + strconv.Itoa(flags.Offset) + "&"
	params += "order=" + flags.Order

	url := flags.APIURL + "/api/memories" + params
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

	var memories []memory.Memory
	if err := json.NewDecoder(resp.Body).Decode(&memories); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if flags.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(memories)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAMESPACE\tCONTENT\tCREATED")
	for _, m := range memories {
		content := m.Content
		if len(content) > 50 {
			content = content[:47] + "..."
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			m.ID.String(), m.Namespace, content, m.CreatedAt.Format("2006-01-02"))
	}
	return w.Flush()
}

func runMemoriesGetCLI(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("memory ID required")
	}

	id := args[0]
	flags, err := parseMemoriesFlags(args[1:])
	if err != nil {
		return err
	}

	return runMemoriesGetWithFlags(ctx, id, flags)
}

func runMemoriesGetWithFlags(ctx context.Context, id string, flags *MemoriesFlags) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid memory ID: %w", err)
	}

	url := flags.APIURL + "/api/memories/" + id
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

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("memory not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var mem memory.Memory
	if err := json.NewDecoder(resp.Body).Decode(&mem); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if flags.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(mem)
	}

	fmt.Printf("ID:        %s\n", mem.ID)
	fmt.Printf("Namespace: %s\n", mem.Namespace)
	fmt.Printf("Content:   %s\n", mem.Content)
	fmt.Printf("Metadata:  %v\n", mem.Metadata)
	fmt.Printf("Created:   %s\n", mem.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:   %s\n", mem.UpdatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

func runMemoriesCreateCLI(ctx context.Context, args []string) error {
	flags, err := parseMemoriesFlags(args)
	if err != nil {
		return err
	}
	return runMemoriesCreateWithFlags(ctx, flags)
}

func runMemoriesCreateWithFlags(ctx context.Context, flags *MemoriesFlags) error {
	content := flags.Content
	if flags.File != "" {
		data, err := os.ReadFile(flags.File)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		content = string(data)
	}

	if content == "" {
		return fmt.Errorf("content required (use --content or --file)")
	}

	metadata := make(map[string]interface{})
	if flags.Metadata != "" {
		parts := strings.SplitN(flags.Metadata, "=", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		}
	}

	reqBody := map[string]interface{}{
		"content":   content,
		"namespace": flags.Namespace,
		"metadata":  metadata,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := flags.APIURL + "/api/memories"
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

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var mem memory.Memory
	if err := json.NewDecoder(resp.Body).Decode(&mem); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if flags.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(mem)
	}

	fmt.Printf("Created memory: %s\n", mem.ID)
	return nil
}

func runMemoriesDeleteCLI(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("memory ID required")
	}

	id := args[0]
	flags, err := parseMemoriesFlags(args[1:])
	if err != nil {
		return err
	}

	return runMemoriesDeleteWithFlags(ctx, id, flags)
}

func runMemoriesDeleteWithFlags(ctx context.Context, id string, flags *MemoriesFlags) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid memory ID: %w", err)
	}

	if !flags.Force {
		fmt.Printf("Delete memory %s? [y/N]: ", id)
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	url := flags.APIURL + "/api/memories/" + id
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
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

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("memory not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Deleted memory: %s\n", id)
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
