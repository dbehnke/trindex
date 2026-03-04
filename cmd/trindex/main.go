package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dbehnke/trindex/internal/cmd"
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

	router := cmd.NewRouter()

	router.Register(cmd.NewMCPCommand())
	router.Register(cmd.NewServerCommand())
	router.Register(cmd.NewDoctorCommand())
	router.Register(cmd.NewMemoriesCommand())
	router.Register(cmd.NewSearchCommand())
	router.Register(cmd.NewStatsCommand())
	router.Register(cmd.NewExportCommand())
	router.Register(cmd.NewImportCommand())

	return router.Run(ctx, os.Args[1:])
}
