package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/mcp"
	"github.com/dbehnke/trindex/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("shutting down...")
		cancel()
	}()

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

	errChan := make(chan error, 2)

	mcpServer := mcp.NewServer(cfg, database, embedClient)
	mcpServer.RegisterTools()

	go func() {
		slog.Info("Trindex MCP server starting", "transport", cfg.Transport)
		if err := mcpServer.Run(ctx); err != nil {
			errChan <- fmt.Errorf("mcp server error: %w", err)
		}
	}()

	webServer := web.NewServer(cfg, database, embedClient)

	go func() {
		if err := webServer.Run(ctx); err != nil {
			errChan <- fmt.Errorf("web server error: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown complete")
		return nil
	case err := <-errChan:
		cancel()
		return err
	}
}
