package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCPCommand(t *testing.T) {
	t.Run("requires valid config", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")

		ctx := context.Background()
		err := RunMCP(ctx, &MCPFlags{})

		assert.Error(t, err)
	})

	t.Run("returns error on invalid database", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://invalid:5432/db")
		t.Setenv("EMBED_BASE_URL", "http://localhost:11434/v1")
		t.Setenv("EMBED_MODEL", "nomic-embed-text")
		t.Setenv("EMBED_DIMENSIONS", "768")

		ctx := context.Background()
		err := RunMCP(ctx, &MCPFlags{})

		assert.Error(t, err)
	})
}

func TestMCFlagsParsing(t *testing.T) {
	flags := &MCPFlags{}

	assert.Equal(t, "", flags.ConfigPath)
	assert.Equal(t, "", flags.RemoteURL)
	assert.Equal(t, "", flags.APIKey)
}
