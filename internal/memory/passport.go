package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ContextPassport struct {
	ID         uuid.UUID `json:"id"`
	Source     string    `json:"source"`
	Target     string    `json:"target"`
	Query      string    `json:"query"`
	TopK       int       `json:"top_k"`
	Memories   []Memory  `json:"memories"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	TTLSeconds int32     `json:"ttl_seconds"`
}

type PassportParams struct {
	Source     string
	Target     string
	Query      string
	Namespaces []string
	TopK       int
	TTLSeconds int32
}

func (s *Store) CreatePassport(ctx context.Context, params PassportParams) (*ContextPassport, error) {
	if params.TopK <= 0 {
		params.TopK = 10
	}
	if params.TTLSeconds <= 0 {
		params.TTLSeconds = 3600
	}

	recallParams := RecallParams{
		Query:      params.Query,
		Namespaces: params.Namespaces,
		TopK:       params.TopK,
		Threshold:  0.3,
	}

	results, err := s.Recall(ctx, recallParams)
	if err != nil {
		return nil, err
	}

	memories := make([]Memory, len(results))
	for i, r := range results {
		memories[i] = r.Memory
	}

	now := time.Now()
	passport := &ContextPassport{
		ID:         uuid.New(),
		Source:     params.Source,
		Target:     params.Target,
		Query:      params.Query,
		TopK:       params.TopK,
		Memories:   memories,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Duration(params.TTLSeconds) * time.Second),
		TTLSeconds: params.TTLSeconds,
	}

	return passport, nil
}

func (s *Store) ImportPassport(ctx context.Context, passport *ContextPassport, targetNamespace string) (int, error) {
	if targetNamespace == "" {
		targetNamespace = "default"
	}

	imported := 0
	for _, mem := range passport.Memories {
		if mem.ExpiresAt != nil && mem.ExpiresAt.Before(time.Now()) {
			continue
		}

		metadata := make(map[string]interface{})
		for k, v := range mem.Metadata {
			metadata[k] = v
		}
		if metadata["imported_from"] == nil {
			metadata["imported_from"] = passport.Source
		}
		if metadata["passport_id"] == nil {
			metadata["passport_id"] = passport.ID.String()
		}

		_, err := s.CreateWithParams(ctx, CreateParams{
			Content:    mem.Content,
			Namespace:  targetNamespace,
			Metadata:   metadata,
			TTLSeconds: mem.TTLSeconds,
		})
		if err != nil {
			continue
		}
		imported++
	}

	return imported, nil
}

type PassportFilter struct {
	Source string
	Target string
	Since  *time.Time
}

func (s *Store) ListPassports(ctx context.Context, filter PassportFilter) ([]*ContextPassport, error) {
	return []*ContextPassport{}, nil
}
