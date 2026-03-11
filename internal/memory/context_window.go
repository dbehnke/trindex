package memory

import (
	"context"
	"math"
	"sort"
	"time"
)

type ContextWindowParams struct {
	Query           string
	Namespaces      []string
	MaxTokens       int
	TokenBudget     int
	RelevanceWeight float64
	RecencyWeight   float64
	TypeBoostWeight float64
}

type RankedMemory struct {
	Memory
	Score     float64
	Relevance float64
	Recency   float64
	TypeBoost float64
}

func (s *Store) BuildContextWindow(ctx context.Context, params ContextWindowParams) ([]RankedMemory, error) {
	if params.MaxTokens <= 0 {
		params.MaxTokens = 4000
	}
	if params.RelevanceWeight == 0 {
		params.RelevanceWeight = 0.5
	}
	if params.RecencyWeight == 0 {
		params.RecencyWeight = 0.3
	}
	if params.TypeBoostWeight == 0 {
		params.TypeBoostWeight = 0.2
	}

	recallParams := RecallParams{
		Query:      params.Query,
		Namespaces: params.Namespaces,
		TopK:       50,
		Threshold:  0.1,
	}

	results, err := s.Recall(ctx, recallParams)
	if err != nil {
		return nil, err
	}

	ranked := s.rankMemories(results)

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	return s.fitToTokenBudget(ranked, params.MaxTokens), nil
}

func (s *Store) rankMemories(results []RecallResult) []RankedMemory {
	if len(results) == 0 {
		return []RankedMemory{}
	}

	now := time.Now()
	oldestTime := results[0].CreatedAt
	newestTime := results[0].CreatedAt

	for _, r := range results {
		if r.CreatedAt.Before(oldestTime) {
			oldestTime = r.CreatedAt
		}
		if r.CreatedAt.After(newestTime) {
			newestTime = r.CreatedAt
		}
	}

	ranked := make([]RankedMemory, len(results))
	for i, r := range results {
		rm := RankedMemory{
			Memory:    r.Memory,
			Relevance: r.Score,
		}

		age := now.Sub(r.CreatedAt).Hours()
		rm.Recency = math.Exp(-age / 168)

		if memType, ok := r.Metadata["type"].(string); ok {
			rm.TypeBoost = typeBoostScore(memType)
		}

		rm.Score = 0.5*rm.Relevance + 0.3*rm.Recency + 0.2*rm.TypeBoost
		ranked[i] = rm
	}

	return ranked
}

func typeBoostScore(memType string) float64 {
	boosts := map[string]float64{
		"preference": 1.0,
		"decision":   0.9,
		"bug":        0.85,
		"pattern":    0.8,
		"outcome":    0.7,
		"fact":       0.6,
	}

	if boost, ok := boosts[memType]; ok {
		return boost
	}
	return 0.5
}

func (s *Store) fitToTokenBudget(ranked []RankedMemory, maxTokens int) []RankedMemory {
	estimatedTokens := 0
	var result []RankedMemory

	for _, rm := range ranked {
		contentTokens := estimateTokens(rm.Content)
		metadataTokens := estimateTokensFromMetadata(rm.Metadata)
		memoryTokens := contentTokens + metadataTokens + 50

		if estimatedTokens+memoryTokens > maxTokens {
			break
		}

		result = append(result, rm)
		estimatedTokens += memoryTokens
	}

	return result
}

func estimateTokens(text string) int {
	return int(math.Ceil(float64(len(text)) / 4.0))
}

func estimateTokensFromMetadata(metadata map[string]interface{}) int {
	if metadata == nil {
		return 0
	}
	estimate := 0
	for k, v := range metadata {
		estimate += estimateTokens(k)
		switch val := v.(type) {
		case string:
			estimate += estimateTokens(val)
		case []interface{}:
			for _, item := range val {
				if s, ok := item.(string); ok {
					estimate += estimateTokens(s)
				}
			}
		}
	}
	return estimate
}
