package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoctorCommand(t *testing.T) {
	t.Run("reports config errors", func(t *testing.T) {
		t.Setenv("TRINDEX_URL", "")
		t.Setenv("EMBED_DIMENSIONS", "0")

		ctx := context.Background()
		exitCode := RunDoctor(ctx, &DoctorFlags{})

		assert.Equal(t, 1, exitCode)
	})

	t.Run("fails when db and embed are unreachable", func(t *testing.T) {
		t.Setenv("TRINDEX_URL", "")
		t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
		t.Setenv("EMBED_BASE_URL", "http://localhost:11434/v1")
		t.Setenv("EMBED_MODEL", "nomic-embed-text")
		t.Setenv("EMBED_DIMENSIONS", "768")

		ctx := context.Background()
		exitCode := RunDoctor(ctx, &DoctorFlags{})

		assert.Equal(t, 1, exitCode)
	})
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"postgres://user:pass@localhost:5432/db",
			"postgres://user:***@localhost:5432/db",
		},
		{
			"postgres://user@localhost:5432/db",
			"postgres://user@localhost:5432/db",
		},
		{
			"localhost:5432/db",
			"localhost:5432/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := maskPassword(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
