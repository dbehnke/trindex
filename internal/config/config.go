package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration
type Config struct {
	// Database
	DatabaseURL string `yaml:"database_url"`

	// Embedding
	EmbedBaseURL    string `yaml:"embed_base_url"`
	EmbedModel      string `yaml:"embed_model"`
	EmbedAPIKey     string `yaml:"embed_api_key"`
	EmbedDimensions int    `yaml:"embed_dimensions"`

	// MCP Transport
	Transport string `yaml:"transport"`

	// HTTP Server (Phase 2)
	HTTPPort    string   `yaml:"http_port"`
	HTTPHost    string   `yaml:"http_host"`
	HTTPAPIKey  string   `yaml:"http_api_key"`
	CORSOrigins []string `yaml:"cors_origins"`

	// HNSW Index Tuning
	HNSWM              int `yaml:"hnsw_m"`
	HNSWEfConstruction int `yaml:"hnsw_ef_construction"`
	HNSWEfSearch       int `yaml:"hnsw_ef_search"`

	// Recall Defaults
	DefaultNamespace           string  `yaml:"default_namespace"`
	DefaultTopK                int     `yaml:"default_top_k"`
	DefaultSimilarityThreshold float64 `yaml:"default_similarity_threshold"`

	// Hybrid Search Weights
	HybridVectorWeight float64 `yaml:"hybrid_vector_weight"`
	HybridFTSWeight    float64 `yaml:"hybrid_fts_weight"`

	// Connection Pooling
	DBMaxConns        int32 `yaml:"db_max_conns"`
	DBMinConns        int32 `yaml:"db_min_conns"`
	DBMaxConnLifetime int   `yaml:"db_max_conn_lifetime_minutes"`
	DBMaxConnIdleTime int   `yaml:"db_max_conn_idle_time_minutes"`

	// Embedding Client Retry
	EmbedMaxRetries     int `yaml:"embed_max_retries"`
	EmbedRetryDelay     int `yaml:"embed_retry_delay_ms"`
	EmbedRequestTimeout int `yaml:"embed_request_timeout_sec"`
}

// LoadWithPath loads configuration from file and environment variables
// Config precedence (highest to lowest):
// 1. Environment variables
// 2. Config file specified by path
// 3. Config file in standard locations
// 4. Default values
func LoadWithPath(configPath string) (*Config, error) {
	cfg := defaultConfig()

	// Try to load from config file
	if configPath != "" {
		// Explicit config path provided
		if err := loadFromFile(cfg, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
		}
	} else {
		// Try standard config locations
		if path := findConfigFile(); path != "" {
			if err := loadFromFile(cfg, path); err != nil {
				// Log but don't fail - env vars might be sufficient
				fmt.Fprintf(os.Stderr, "Warning: failed to load config from %s: %v\n", path, err)
			}
		}
	}

	// Environment variables override config file
	cfg.loadFromEnv()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Load loads configuration using the standard config locations
func Load() (*Config, error) {
	return LoadWithPath("")
}

// defaultConfig returns a Config with default values
func defaultConfig() *Config {
	return &Config{
		// Database
		DatabaseURL: "postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable",

		// Embedding
		EmbedBaseURL:    "http://localhost:11434/v1",
		EmbedModel:      "nomic-embed-text",
		EmbedAPIKey:     "ollama",
		EmbedDimensions: 768,

		// MCP Transport
		Transport: "stdio",

		// HTTP Server
		HTTPPort:    "9636",
		HTTPHost:    "0.0.0.0",
		HTTPAPIKey:  "",
		CORSOrigins: []string{"http://localhost:5173", "http://localhost:9636"},

		// HNSW Index Tuning
		HNSWM:              16,
		HNSWEfConstruction: 64,
		HNSWEfSearch:       40,

		// Recall Defaults
		DefaultNamespace:           "default",
		DefaultTopK:                10,
		DefaultSimilarityThreshold: 0.7,

		// Hybrid Search Weights
		HybridVectorWeight: 0.7,
		HybridFTSWeight:    0.3,

		// Connection Pooling
		DBMaxConns:        100,
		DBMinConns:        10,
		DBMaxConnLifetime: 60,
		DBMaxConnIdleTime: 30,

		// Embedding Client Retry
		EmbedMaxRetries:     3,
		EmbedRetryDelay:     1000,
		EmbedRequestTimeout: 30,
	}
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}

// loadFromEnv overrides config values from environment variables
func (c *Config) loadFromEnv() {
	// Database
	if v := os.Getenv("DATABASE_URL"); v != "" {
		c.DatabaseURL = v
	}

	// Embedding
	if v := os.Getenv("EMBED_BASE_URL"); v != "" {
		c.EmbedBaseURL = v
	}
	if v := os.Getenv("EMBED_MODEL"); v != "" {
		c.EmbedModel = v
	}
	if v := os.Getenv("EMBED_API_KEY"); v != "" {
		c.EmbedAPIKey = v
	}
	if v := os.Getenv("EMBED_DIMENSIONS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.EmbedDimensions = val
		}
	}

	// MCP Transport
	if v := os.Getenv("TRANSPORT"); v != "" {
		c.Transport = v
	}

	// HTTP Server
	if v := os.Getenv("HTTP_PORT"); v != "" {
		c.HTTPPort = v
	}
	if v := os.Getenv("HTTP_HOST"); v != "" {
		c.HTTPHost = v
	}
	if v := os.Getenv("TRINDEX_API_KEY"); v != "" {
		c.HTTPAPIKey = v
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		// allow comma separated list of origins in the env variable
		c.CORSOrigins = strings.Split(v, ",")
		for i, o := range c.CORSOrigins {
			c.CORSOrigins[i] = strings.TrimSpace(o)
		}
	}

	// HNSW Index Tuning
	if v := os.Getenv("HNSW_M"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.HNSWM = val
		}
	}
	if v := os.Getenv("HNSW_EF_CONSTRUCTION"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.HNSWEfConstruction = val
		}
	}
	if v := os.Getenv("HNSW_EF_SEARCH"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.HNSWEfSearch = val
		}
	}

	// Recall Defaults
	if v := os.Getenv("DEFAULT_NAMESPACE"); v != "" {
		c.DefaultNamespace = v
	}
	if v := os.Getenv("DEFAULT_TOP_K"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.DefaultTopK = val
		}
	}
	if v := os.Getenv("DEFAULT_SIMILARITY_THRESHOLD"); v != "" {
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			c.DefaultSimilarityThreshold = val
		}
	}

	// Hybrid Search Weights
	if v := os.Getenv("HYBRID_VECTOR_WEIGHT"); v != "" {
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			c.HybridVectorWeight = val
		}
	}
	if v := os.Getenv("HYBRID_FTS_WEIGHT"); v != "" {
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			c.HybridFTSWeight = val
		}
	}

	// Connection Pooling
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.DBMaxConns = int32(val)
		}
	}
	if v := os.Getenv("DB_MIN_CONNS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.DBMinConns = int32(val)
		}
	}
	if v := os.Getenv("DB_MAX_CONN_LIFETIME_MINUTES"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.DBMaxConnLifetime = val
		}
	}
	if v := os.Getenv("DB_MAX_CONN_IDLE_TIME_MINUTES"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.DBMaxConnIdleTime = val
		}
	}

	// Embedding Client Retry
	if v := os.Getenv("EMBED_MAX_RETRIES"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.EmbedMaxRetries = val
		}
	}
	if v := os.Getenv("EMBED_RETRY_DELAY_MS"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.EmbedRetryDelay = val
		}
	}
	if v := os.Getenv("EMBED_REQUEST_TIMEOUT_SEC"); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			c.EmbedRequestTimeout = val
		}
	}
}

// findConfigFile searches for config file in standard locations
// Returns empty string if no config file is found
func findConfigFile() string {
	// Check TRINDEX_CONFIG env var first
	if path := os.Getenv("TRINDEX_CONFIG"); path != "" {
		if fileExists(path) {
			return path
		}
	}

	// Get user config directory (XDG Base Directory spec)
	configDir := getUserConfigDir()

	// Search paths in order of precedence
	searchPaths := []string{
		// Current directory
		"trindex.yaml",
		".trindex.yaml",
		// XDG config directory
		filepath.Join(configDir, "trindex", "config.yaml"),
		filepath.Join(configDir, "trindex", "trindex.yaml"),
		// Legacy locations
		filepath.Join(getHomeDir(), ".trindex.yaml"),
		filepath.Join(getHomeDir(), ".trindex", "config.yaml"),
	}

	// System-wide config (Unix only)
	if runtime.GOOS != "windows" {
		searchPaths = append(searchPaths,
			"/etc/trindex/config.yaml",
			"/etc/trindex.yaml",
		)
	}

	for _, path := range searchPaths {
		if fileExists(path) {
			return path
		}
	}

	return ""
}

// getUserConfigDir returns the user's config directory following XDG spec
func getUserConfigDir() string {
	// Check XDG_CONFIG_HOME
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return xdgConfig
	}

	// Fallback to platform-specific defaults
	home := getHomeDir()

	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support
		return filepath.Join(home, "Library", "Application Support")
	case "windows":
		// Windows: %APPDATA% or %LOCALAPPDATA%
		if appData := os.Getenv("APPDATA"); appData != "" {
			return appData
		}
		return filepath.Join(home, "AppData", "Roaming")
	default:
		// Linux and other Unix: ~/.config
		return filepath.Join(home, ".config")
	}
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback
		if runtime.GOOS == "windows" {
			return os.Getenv("USERPROFILE")
		}
		return os.Getenv("HOME")
	}
	return home
}

// fileExists checks if a file exists and is readable
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Validate checks that all configuration is valid
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.EmbedBaseURL == "" {
		return fmt.Errorf("EMBED_BASE_URL is required")
	}

	if c.EmbedModel == "" {
		return fmt.Errorf("EMBED_MODEL is required")
	}

	if c.EmbedDimensions <= 0 {
		return fmt.Errorf("EMBED_DIMENSIONS must be positive, got %d", c.EmbedDimensions)
	}

	if c.Transport != "stdio" && c.Transport != "http" {
		return fmt.Errorf("TRANSPORT must be 'stdio' or 'http', got '%s'", c.Transport)
	}

	if c.HNSWM <= 0 {
		return fmt.Errorf("HNSW_M must be positive")
	}

	if c.HNSWEfConstruction <= 0 {
		return fmt.Errorf("HNSW_EF_CONSTRUCTION must be positive")
	}

	if c.HNSWEfSearch <= 0 {
		return fmt.Errorf("HNSW_EF_SEARCH must be positive")
	}

	if c.DefaultTopK <= 0 {
		return fmt.Errorf("DEFAULT_TOP_K must be positive")
	}

	if c.DefaultSimilarityThreshold < 0 || c.DefaultSimilarityThreshold > 1 {
		return fmt.Errorf("DEFAULT_SIMILARITY_THRESHOLD must be between 0 and 1")
	}

	if c.HybridVectorWeight < 0 || c.HybridVectorWeight > 1 {
		return fmt.Errorf("HYBRID_VECTOR_WEIGHT must be between 0 and 1")
	}

	if c.HybridFTSWeight < 0 || c.HybridFTSWeight > 1 {
		return fmt.Errorf("HYBRID_FTS_WEIGHT must be between 0 and 1")
	}

	return nil
}

// ConfigPath returns the path to the currently used config file
// or empty string if using defaults
func ConfigPath() string {
	return findConfigFile()
}

// DefaultConfigPath returns the recommended config file path
// for creating a new config file
func DefaultConfigPath() string {
	return filepath.Join(getUserConfigDir(), "trindex", "config.yaml")
}
