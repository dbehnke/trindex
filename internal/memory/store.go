package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type Store struct {
	db    *db.DB
	embed *embed.Client
	cfg   *config.Config
}

func NewStore(database *db.DB, embedClient *embed.Client, cfg *config.Config) *Store {
	return &Store{
		db:    database,
		embed: embedClient,
		cfg:   cfg,
	}
}

func (s *Store) Create(ctx context.Context, content, namespace string, metadata map[string]interface{}) (*Memory, error) {
	return s.CreateWithParams(ctx, CreateParams{
		Content:   content,
		Namespace: namespace,
		Metadata:  metadata,
	})
}

func (s *Store) CreateWithParams(ctx context.Context, params CreateParams) (*Memory, error) {
	if params.Namespace == "" {
		params.Namespace = "default"
	}
	if params.Metadata == nil {
		params.Metadata = make(map[string]interface{})
	}

	contentHash := computeContentHash(params.Content)

	if existing, err := s.findByContentHash(ctx, params.Namespace, contentHash); err == nil && existing != nil {
		return existing, nil
	}

	embedding, err := s.embed.Embed(params.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	now := time.Now()
	var expiresAt *time.Time
	if params.TTLSeconds > 0 {
		exp := now.Add(time.Duration(params.TTLSeconds) * time.Second)
		expiresAt = &exp
	}

	memory := &Memory{
		ID:          uuid.New(),
		Namespace:   params.Namespace,
		Content:     params.Content,
		ContentHash: contentHash,
		Embedding:   pgvector.NewVector(embedding),
		Metadata:    params.Metadata,
		TTLSeconds:  params.TTLSeconds,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	query := `
		INSERT INTO memories (id, namespace, content, content_hash, embedding, metadata, ttl_seconds, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = s.db.Pool().Exec(ctx, query,
		memory.ID, memory.Namespace, memory.Content, memory.ContentHash, memory.Embedding,
		memory.Metadata, memory.TTLSeconds, memory.ExpiresAt, memory.CreatedAt, memory.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert memory: %w", err)
	}

	return memory, nil
}

func computeContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

func (s *Store) findByContentHash(ctx context.Context, namespace, contentHash string) (*Memory, error) {
	query := `
		SELECT id, namespace, content, content_hash, metadata, ttl_seconds, expires_at, created_at, updated_at
		FROM memories
		WHERE namespace = $1 AND content_hash = $2
	`

	var m Memory
	var expiresAt *time.Time
	var ttlSeconds *int32
	err := s.db.Pool().QueryRow(ctx, query, namespace, contentHash).Scan(
		&m.ID, &m.Namespace, &m.Content, &m.ContentHash,
		&m.Metadata, &ttlSeconds, &expiresAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.ExpiresAt = expiresAt
	if ttlSeconds != nil {
		m.TTLSeconds = *ttlSeconds
	}

	if m.ExpiresAt != nil && m.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	return &m, nil
}

func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (*Memory, error) {
	query := `
		SELECT id, namespace, content, content_hash, metadata, ttl_seconds, expires_at, created_at, updated_at
		FROM memories
		WHERE id = $1
	`

	var m Memory
	var expiresAt *time.Time
	var ttlSeconds *int32
	err := s.db.Pool().QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Namespace, &m.Content, &m.ContentHash,
		&m.Metadata, &ttlSeconds, &expiresAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("memory not found: %w", err)
	}
	m.ExpiresAt = expiresAt
	if ttlSeconds != nil {
		m.TTLSeconds = *ttlSeconds
	}

	return &m, nil
}

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

func (s *Store) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM memories WHERE expires_at IS NOT NULL AND expires_at < NOW()`
	result, err := s.db.Pool().Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired memories: %w", err)
	}

	return result.RowsAffected(), nil
}

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

	query := `SELECT id, namespace, content, content_hash, metadata, ttl_seconds, expires_at, created_at, updated_at FROM memories`
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
		var expiresAt *time.Time
		var ttlSeconds *int32
		err := rows.Scan(&m.ID, &m.Namespace, &m.Content, &m.ContentHash,
			&m.Metadata, &ttlSeconds, &expiresAt, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		m.ExpiresAt = expiresAt
		if ttlSeconds != nil {
			m.TTLSeconds = *ttlSeconds
		}
		memories = append(memories, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return memories, nil
}
