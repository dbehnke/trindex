package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// Database
	DatabaseURL string

	// Embedding
	EmbedBaseURL    string
	EmbedModel      string
	EmbedAPIKey     string
	EmbedDimensions int

	// MCP Transport
	Transport string

	// HTTP Server (Phase 2)
	HTTPPort   string
	HTTPHost   string
	HTTPAPIKey string

	// HNSW Index Tuning
	HNSWM              int
	HNSWEfConstruction int
	HNSWEfSearch       int

	// Recall Defaults
	DefaultNamespace           string
	DefaultTopK                int
	DefaultSimilarityThreshold float64

	// Hybrid Search Weights
	HybridVectorWeight float64
	HybridFTSWeight    float64

	// Connection Pooling
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime int
	DBMaxConnIdleTime int

	// Embedding Client Retry
	EmbedMaxRetries     int
	EmbedRetryDelay     int
	EmbedRequestTimeout int
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable"),

		// Embedding
		EmbedBaseURL:    getEnv("EMBED_BASE_URL", "http://localhost:11434/v1"),
		EmbedModel:      getEnv("EMBED_MODEL", "nomic-embed-text"),
		EmbedAPIKey:     getEnv("EMBED_API_KEY", "ollama"),
		EmbedDimensions: getEnvAsInt("EMBED_DIMENSIONS", 768),

		// MCP Transport
		Transport: getEnv("TRANSPORT", "stdio"),

		// HTTP Server
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		HTTPHost:   getEnv("HTTP_HOST", "0.0.0.0"),
		HTTPAPIKey: getEnv("TRINDEX_API_KEY", ""),

		// HNSW Index Tuning
		HNSWM:              getEnvAsInt("HNSW_M", 16),
		HNSWEfConstruction: getEnvAsInt("HNSW_EF_CONSTRUCTION", 64),
		HNSWEfSearch:       getEnvAsInt("HNSW_EF_SEARCH", 40),

		// Recall Defaults
		DefaultNamespace:           getEnv("DEFAULT_NAMESPACE", "default"),
		DefaultTopK:                getEnvAsInt("DEFAULT_TOP_K", 10),
		DefaultSimilarityThreshold: getEnvAsFloat("DEFAULT_SIMILARITY_THRESHOLD", 0.7),

		// Hybrid Search Weights
		HybridVectorWeight: getEnvAsFloat("HYBRID_VECTOR_WEIGHT", 0.7),
		HybridFTSWeight:    getEnvAsFloat("HYBRID_FTS_WEIGHT", 0.3),

		// Connection Pooling
		DBMaxConns:        int32(getEnvAsInt("DB_MAX_CONNS", 100)),
		DBMinConns:        int32(getEnvAsInt("DB_MIN_CONNS", 10)),
		DBMaxConnLifetime: getEnvAsInt("DB_MAX_CONN_LIFETIME_MINUTES", 60),
		DBMaxConnIdleTime: getEnvAsInt("DB_MAX_CONN_IDLE_TIME_MINUTES", 30),

		// Embedding Client Retry
		EmbedMaxRetries:     getEnvAsInt("EMBED_MAX_RETRIES", 3),
		EmbedRetryDelay:     getEnvAsInt("EMBED_RETRY_DELAY_MS", 1000),
		EmbedRequestTimeout: getEnvAsInt("EMBED_REQUEST_TIMEOUT_SEC", 30),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
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

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return defaultValue
	}
	return value
}
