package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	assert.Equal(t, "postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "http://localhost:11434/v1", cfg.EmbedBaseURL)
	assert.Equal(t, "nomic-embed-text", cfg.EmbedModel)
	assert.Equal(t, 768, cfg.EmbedDimensions)
	assert.Equal(t, "stdio", cfg.Transport)
	assert.Equal(t, "8080", cfg.HTTPPort)
	assert.Equal(t, "default", cfg.DefaultNamespace)
	assert.Equal(t, 10, cfg.DefaultTopK)
}

func TestConfigFromEnv(t *testing.T) {
	// Set test env vars
	_ = os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	_ = os.Setenv("EMBED_MODEL", "test-model")
	_ = os.Setenv("DEFAULT_TOP_K", "20")
	defer func() {
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("EMBED_MODEL")
		_ = os.Unsetenv("DEFAULT_TOP_K")
	}()

	cfg := defaultConfig()
	cfg.loadFromEnv()

	assert.Equal(t, "postgres://test:test@localhost:5432/test", cfg.DatabaseURL)
	assert.Equal(t, "test-model", cfg.EmbedModel)
	assert.Equal(t, 20, cfg.DefaultTopK)
}

func TestConfigFromFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database_url: "postgres://file:test@localhost/filedb"
embed_model: "file-model"
default_top_k: 50
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg := defaultConfig()
	err = loadFromFile(cfg, configPath)
	require.NoError(t, err)

	assert.Equal(t, "postgres://file:test@localhost/filedb", cfg.DatabaseURL)
	assert.Equal(t, "file-model", cfg.EmbedModel)
	assert.Equal(t, 50, cfg.DefaultTopK)
}

func TestLoadWithPath(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
database_url: "postgres://explicit:test@localhost/db"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load with explicit path
	cfg, err := LoadWithPath(configPath)
	require.NoError(t, err)

	assert.Equal(t, "postgres://explicit:test@localhost/db", cfg.DatabaseURL)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "missing database_url",
			modify: func(c *Config) {
				c.DatabaseURL = ""
			},
			wantErr: true,
		},
		{
			name: "invalid dimensions",
			modify: func(c *Config) {
				c.EmbedDimensions = 0
			},
			wantErr: true,
		},
		{
			name: "invalid threshold high",
			modify: func(c *Config) {
				c.DefaultSimilarityThreshold = 1.5
			},
			wantErr: true,
		},
		{
			name: "invalid threshold low",
			modify: func(c *Config) {
				c.DefaultSimilarityThreshold = -0.1
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindConfigFile(t *testing.T) {
	// Test with no config files (should return empty)
	path := findConfigFile()
	// Result depends on system, just verify it doesn't panic
	_ = path
}

func TestGetUserConfigDir(t *testing.T) {
	dir := getUserConfigDir()
	assert.NotEmpty(t, dir)

	// Verify it contains the expected path component based on OS
	switch runtime.GOOS {
	case "darwin":
		assert.Contains(t, dir, "Application Support")
	case "windows":
		// On Windows, could be various paths
		assert.NotEmpty(t, dir)
	default:
		assert.Contains(t, dir, ".config")
	}
}

func TestGetHomeDir(t *testing.T) {
	home := getHomeDir()
	assert.NotEmpty(t, home)
	assert.True(t, filepath.IsAbs(home))
}

func TestFileExists(t *testing.T) {
	// Test with existing file
	tmpFile := filepath.Join(t.TempDir(), "exists.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	require.NoError(t, err)

	assert.True(t, fileExists(tmpFile))

	// Test with non-existent file
	assert.False(t, fileExists("/nonexistent/path/file.txt"))

	// Test with directory (should return false)
	assert.False(t, fileExists(t.TempDir()))
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "trindex")
	assert.Contains(t, path, "config.yaml")
}
