package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ExportMemory represents a memory in export format
type ExportMemory struct {
	ID        uuid.UUID              `json:"id"`
	Namespace string                 `json:"namespace"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ExportResult contains export statistics
type ExportResult struct {
	Count     int64  `json:"count"`
	Namespace string `json:"namespace,omitempty"`
}

// Export exports memories to a writer in JSON format
func (s *Store) Export(ctx context.Context, namespace string, since, until *time.Time, w io.Writer) (*ExportResult, error) {
	query := `
		SELECT id, namespace, content, metadata, created_at, updated_at 
		FROM memories 
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIdx)
		args = append(args, namespace)
		argIdx++
	}

	if since != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *since)
		argIdx++
	}

	if until != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *until)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	encoder := json.NewEncoder(w)
	var count int64

	for rows.Next() {
		var m ExportMemory
		err := rows.Scan(&m.ID, &m.Namespace, &m.Content, &m.Metadata, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}

		if err := encoder.Encode(m); err != nil {
			return nil, fmt.Errorf("failed to encode memory: %w", err)
		}
		count++
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &ExportResult{
		Count:     count,
		Namespace: namespace,
	}, nil
}

// ImportMemory represents a memory in import format
type ImportMemory struct {
	ID        uuid.UUID              `json:"id"`
	Namespace string                 `json:"namespace"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt *time.Time             `json:"updated_at,omitempty"`
}

// ImportResult contains import statistics
type ImportResult struct {
	Imported int64    `json:"imported"`
	Failed   int64    `json:"failed"`
	Errors   []string `json:"errors,omitempty"`
}

// Import imports memories from a JSON reader
// Supports OpenBrain format (with some transformations)
func (s *Store) Import(ctx context.Context, r io.Reader, options ImportOptions) (*ImportResult, error) {
	result := &ImportResult{
		Errors: []string{},
	}

	decoder := json.NewDecoder(r)

	for {
		var mem ImportMemory
		if err := decoder.Decode(&mem); err == io.EOF {
			break
		} else if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("decode error: %v", err))
			continue
		}

		if mem.Content == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("empty content for ID %s", mem.ID))
			continue
		}

		namespace := mem.Namespace
		if namespace == "" {
			namespace = options.DefaultNamespace
		}
		if namespace == "" {
			namespace = "default"
		}

		if options.SkipExisting {
			exists, err := s.memoryExists(ctx, mem.ID)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("existence check failed for %s: %v", mem.ID, err))
				continue
			}
			if exists {
				continue
			}
		}

		if options.RegenerateEmbeddings {
			_, err := s.Create(ctx, mem.Content, namespace, mem.Metadata)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("create failed for %s: %v", mem.ID, err))
				continue
			}
		} else {
			err := s.importWithEmbedding(ctx, mem, namespace)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("import failed for %s: %v", mem.ID, err))
				continue
			}
		}

		result.Imported++
	}

	return result, nil
}

// ImportOptions contains options for import
type ImportOptions struct {
	DefaultNamespace     string
	SkipExisting         bool
	RegenerateEmbeddings bool
}

func (s *Store) memoryExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM memories WHERE id = $1)`
	err := s.db.Pool().QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

func (s *Store) importWithEmbedding(ctx context.Context, mem ImportMemory, namespace string) error {
	embedding, err := s.embed.Embed(mem.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	if mem.ID == uuid.Nil {
		mem.ID = uuid.New()
	}

	if mem.Metadata == nil {
		mem.Metadata = make(map[string]interface{})
	}

	now := time.Now()
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = now
	}

	updatedAt := now
	if mem.UpdatedAt != nil {
		updatedAt = *mem.UpdatedAt
	}

	query := `
		INSERT INTO memories (id, namespace, content, embedding, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			namespace = EXCLUDED.namespace,
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`

	_, err = s.db.Pool().Exec(ctx, query,
		mem.ID, namespace, mem.Content, pgvector.NewVector(embedding),
		mem.Metadata, mem.CreatedAt, updatedAt,
	)
	return err
}

// DuplicateCandidate represents a potential duplicate memory
type DuplicateCandidate struct {
	MemoryID         uuid.UUID `json:"memory_id"`
	MemoryContent    string    `json:"memory_content"`
	DuplicateID      uuid.UUID `json:"duplicate_id"`
	DuplicateContent string    `json:"duplicate_content"`
	Similarity       float64   `json:"similarity"`
}

// FindDuplicates finds memories that are near-duplicates of each other
func (s *Store) FindDuplicates(ctx context.Context, namespace string, threshold float64, limit int) ([]DuplicateCandidate, error) {
	if threshold == 0 {
		threshold = 0.95
	}
	if limit == 0 {
		limit = 100
	}

	query := `
		SELECT 
			m1.id, m1.content,
			m2.id, m2.content,
			1 - (m1.embedding <=> m2.embedding) as similarity
		FROM memories m1
		JOIN memories m2 ON m1.id < m2.id
		WHERE 1 - (m1.embedding <=> m2.embedding) >= $1
	`
	args := []interface{}{threshold}
	argIdx := 2

	if namespace != "" {
		query += fmt.Sprintf(" AND m1.namespace = $%d AND m2.namespace = $%d", argIdx, argIdx)
		args = append(args, namespace)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY similarity DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find duplicates: %w", err)
	}
	defer rows.Close()

	var candidates []DuplicateCandidate
	for rows.Next() {
		var c DuplicateCandidate
		err := rows.Scan(&c.MemoryID, &c.MemoryContent, &c.DuplicateID, &c.DuplicateContent, &c.Similarity)
		if err != nil {
			return nil, fmt.Errorf("failed to scan duplicate: %w", err)
		}
		candidates = append(candidates, c)
	}

	return candidates, rows.Err()
}

// MergeDuplicates merges two duplicate memories, keeping the newer one and deleting the older
func (s *Store) MergeDuplicates(ctx context.Context, keepID, removeID uuid.UUID) error {
	tx, err := s.db.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM memories WHERE id = $1`
	_, err = tx.Exec(ctx, query, removeID)
	if err != nil {
		return fmt.Errorf("failed to remove duplicate: %w", err)
	}

	return tx.Commit(ctx)
}
