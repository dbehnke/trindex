package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

// Store handles memory CRUD operations
type Store struct {
	db    *db.DB
	embed *embed.Client
	cfg   *config.Config
}

// NewStore creates a new memory store
func NewStore(database *db.DB, embedClient *embed.Client, cfg *config.Config) *Store {
	return &Store{
		db:    database,
		embed: embedClient,
		cfg:   cfg,
	}
}

// Create stores a new memory with embedding
func (s *Store) Create(ctx context.Context, content, namespace string, metadata map[string]interface{}) (*Memory, error) {
	// Generate embedding
	embedding, err := s.embed.Embed(content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Ensure namespace is set
	if namespace == "" {
		namespace = "default"
	}

	// Ensure metadata is not nil
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	memory := &Memory{
		ID:        uuid.New(),
		Namespace: namespace,
		Content:   content,
		Embedding: pgvector.NewVector(embedding),
		Metadata:  metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO memories (id, namespace, content, embedding, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = s.db.Pool().Exec(ctx, query,
		memory.ID, memory.Namespace, memory.Content, memory.Embedding,
		memory.Metadata, memory.CreatedAt, memory.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert memory: %w", err)
	}

	return memory, nil
}

// GetByID retrieves a single memory by ID
func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (*Memory, error) {
	query := `
		SELECT id, namespace, content, metadata, created_at, updated_at 
		FROM memories 
		WHERE id = $1
	`

	var m Memory
	err := s.db.Pool().QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Namespace, &m.Content,
		&m.Metadata, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("memory not found: %w", err)
	}

	return &m, nil
}

// DeleteByID deletes a single memory by ID
func (s *Store) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM memories WHERE id = $1`
	result, err := s.db.Pool().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}

	return nil
}

// DeleteByNamespace deletes memories by namespace with optional filter
func (s *Store) DeleteByNamespace(ctx context.Context, namespace string, filter ForgetFilter) (int64, error) {
	query := `DELETE FROM memories WHERE namespace = $1`
	args := []interface{}{namespace}
	argIdx := 2

	if filter.Before != nil {
		query += fmt.Sprintf(" AND created_at < $%d", argIdx)
		args = append(args, *filter.Before)
		argIdx++
	}

	if len(filter.Tags) > 0 {
		tagStrs := make([]string, len(filter.Tags))
		for i, t := range filter.Tags {
			tagStrs[i] = `"` + strings.ReplaceAll(t, `"`, `\"`) + `"`
		}
		query += fmt.Sprintf(" AND metadata->'tags' @> $%d::jsonb", argIdx)
		args = append(args, "["+strings.Join(tagStrs, ",")+"]")
	}

	result, err := s.db.Pool().Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete memories: %w", err)
	}

	return result.RowsAffected(), nil
}

// List retrieves memories without semantic search
func (s *Store) List(ctx context.Context, params ListParams) ([]Memory, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Order != "asc" {
		params.Order = "desc"
	}

	query := `SELECT id, namespace, content, metadata, created_at, updated_at FROM memories`
	args := []interface{}{}
	argIdx := 1

	if params.Namespace != "" {
		query += fmt.Sprintf(" WHERE namespace = $%d", argIdx)
		args = append(args, params.Namespace)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at %s", params.Order)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

func scanMemories(rows pgx.Rows) ([]Memory, error) {
	var memories []Memory
	for rows.Next() {
		var m Memory
		err := rows.Scan(&m.ID, &m.Namespace, &m.Content,
			&m.Metadata, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return memories, nil
}
