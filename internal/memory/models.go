package memory

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type Memory struct {
	ID          uuid.UUID              `db:"id" json:"id"`
	Namespace   string                 `db:"namespace" json:"namespace"`
	Content     string                 `db:"content" json:"content"`
	ContentHash string                 `db:"content_hash" json:"content_hash,omitempty"`
	Embedding   pgvector.Vector        `db:"embedding" json:"-"`
	Metadata    map[string]interface{} `db:"metadata" json:"metadata"`
	TTLSeconds  int32                  `db:"ttl_seconds" json:"ttl_seconds,omitempty"`
	ExpiresAt   *time.Time             `db:"expires_at" json:"expires_at,omitempty"`
	CreatedAt   time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `db:"updated_at" json:"updated_at"`
}

// RecallResult represents a memory retrieved by search
type RecallResult struct {
	Memory
	Score float64 `json:"score"`
}

// Filter represents filters for memory queries
type Filter struct {
	Since  *time.Time
	Until  *time.Time
	Tags   []string
	Source string
}

type CreateParams struct {
	Content            string
	Namespace          string
	Metadata           map[string]interface{}
	SkipIfDuplicate    bool
	DuplicateThreshold float64
	TTLSeconds         int32
}

// RecallParams represents parameters for recall operation
type RecallParams struct {
	Query      string
	Namespaces []string
	TopK       int
	Threshold  float64
	Filter     Filter
	// Hybrid search weights (optional, uses config defaults if zero)
	VectorWeight float64
	FTSWeight    float64
}

// ListParams represents parameters for list operation
type ListParams struct {
	Namespace string
	Limit     int
	Offset    int
	Order     string
}

// ForgetFilter represents filters for forget operation
type ForgetFilter struct {
	Before *time.Time
	Tags   []string
}

// Stats represents memory statistics
type Stats struct {
	TotalMemories   int64            `json:"total_memories"`
	ByNamespace     map[string]int64 `json:"by_namespace"`
	Recent24h       int64            `json:"recent_24h"`
	OldestMemory    *time.Time       `json:"oldest_memory,omitempty"`
	NewestMemory    *time.Time       `json:"newest_memory,omitempty"`
	TopTags         []string         `json:"top_tags"`
	EmbeddingModel  string           `json:"embedding_model"`
	EmbedDimensions int              `json:"embed_dimensions"`
}
