// Package testutil provides utilities for integration testing with Testcontainers.
package testutil

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers postgres container for testing.
type PostgresContainer struct {
	Container testcontainers.Container
	ConnStr   string
}

// NewPostgresContainer starts a new pgvector container for testing.
func NewPostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	container, err := postgres.Run(ctx,
		"pgvector/pgvector:pg17",
		postgres.WithDatabase("trindex_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		if termErr := container.Terminate(ctx); termErr != nil {
			return nil, fmt.Errorf("failed to get connection string (and failed to terminate container: %v): %w", termErr, err)
		}
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &PostgresContainer{
		Container: container,
		ConnStr:   connStr,
	}, nil
}

// Terminate stops and removes the container.
func (p *PostgresContainer) Terminate(ctx context.Context) error {
	if p.Container != nil {
		return p.Container.Terminate(ctx)
	}
	return nil
}

// SkipIfNoDocker skips the test if Docker is not available.
// On macOS, it automatically configures Colima environment variables if needed.
func SkipIfNoDocker(t *testing.T) {
	t.Helper()

	if IsCI() {
		return
	}

	// On macOS, configure Colima environment if DOCKER_HOST is not set
	if runtime.GOOS == "darwin" && os.Getenv("DOCKER_HOST") == "" {
		colimaSocket := "unix://" + os.Getenv("HOME") + "/.colima/default/docker.sock"
		if _, err := os.Stat(colimaSocket[7:]); err == nil {
			_ = os.Setenv("DOCKER_HOST", colimaSocket)
			_ = os.Setenv("TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "/var/run/docker.sock")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := testcontainers.NewDockerClientWithOpts(ctx)
	if err != nil {
		if runtime.GOOS == "darwin" {
			t.Skip("Docker not available. If using Colima, ensure it's running: colima start")
		}
		t.Skip("Docker not available:", err)
	}

	_, err = client.Ping(ctx)
	if err != nil {
		if runtime.GOOS == "darwin" {
			t.Skip("Docker not available. If using Colima, ensure it's running: colima start")
		}
		t.Skip("Docker not available:", err)
	}
}

// IsCI returns true if running in a CI environment.
func IsCI() bool {
	return os.Getenv("CI") == "true"
}
