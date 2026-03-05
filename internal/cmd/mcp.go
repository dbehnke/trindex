package cmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/mcp"
)

type MCPFlags struct {
	ConfigPath string
	RemoteURL  string
	APIKey     string
}

func NewMCPCommand() *Command {
	flags := &MCPFlags{}
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	fs.StringVar(&flags.ConfigPath, "config", "", "Config file path")
	fs.StringVar(&flags.RemoteURL, "remote", "", "Remote Trindex HTTP API URL (default: http://localhost:9636)")
	fs.StringVar(&flags.APIKey, "api-key", "", "API key for remote connection (default: TRINDEX_API_KEY env)")

	return &Command{
		Name:        "mcp",
		Description: "Run MCP server (stdio) - proxies to remote Trindex server",
		Flags:       fs,
		Run: func(ctx context.Context, args []string) error {
			return RunMCP(ctx, flags)
		},
	}
}

func RunMCP(ctx context.Context, flags *MCPFlags) error {
	serverURL := flags.RemoteURL
	if serverURL == "" {
		serverURL = os.Getenv("TRINDEX_URL")
	}
	if serverURL == "" {
		serverURL = "http://localhost:9636"
	}

	apiKey := flags.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("TRINDEX_API_KEY")
	}

	if serverURL != "" && serverURL != "local" {
		slog.Info("Trindex MCP client starting in proxy mode", "server", serverURL)
		return RunMCPProxy(ctx, serverURL, apiKey)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	if err := database.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	embedClient := embed.NewClient(cfg)
	if err := embedClient.ValidateDimensions(); err != nil {
		return fmt.Errorf("embedding validation failed: %w", err)
	}

	mcpServer := mcp.NewServer(cfg, database, embedClient)
	mcpServer.RegisterTools()

	slog.Info("Trindex MCP server starting in local mode", "transport", "stdio")
	if err := mcpServer.Run(ctx); err != nil {
		return fmt.Errorf("mcp server error: %w", err)
	}

	return nil
}

func RunMCPProxy(ctx context.Context, serverURL, apiKey string) error {
	proxy := NewMCPProxy(serverURL, apiKey)
	return proxy.Run(ctx)
}
