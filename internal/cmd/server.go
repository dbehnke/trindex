package cmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/web"
)

type ServerFlags struct {
	Host string
	Port string
	NoUI bool
}

func NewServerCommand() *Command {
	flags := &ServerFlags{}
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	fs.StringVar(&flags.Host, "host", "", "HTTP host (default: 0.0.0.0)")
	fs.StringVar(&flags.Port, "port", "", "HTTP port (default from config: 9636)")
	fs.BoolVar(&flags.NoUI, "no-ui", false, "Disable web UI, API only")

	return &Command{
		Name:        "server",
		Description: "Run HTTP server only",
		Flags:       fs,
		Run: func(ctx context.Context, args []string) error {
			return RunServer(ctx, flags)
		},
	}
}

func RunServer(ctx context.Context, flags *ServerFlags) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if flags.Host != "" {
		cfg.HTTPHost = flags.Host
	}
	if flags.Port != "" {
		cfg.HTTPPort = flags.Port
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

	webServer := web.NewServer(cfg, database, embedClient)

	slog.Info("Trindex HTTP server starting",
		"addr", fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort))

	if err := webServer.Run(ctx); err != nil {
		return fmt.Errorf("web server error: %w", err)
	}

	return nil
}
