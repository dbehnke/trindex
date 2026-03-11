package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// Recall performs hybrid search combining vector similarity and full-text search
func (s *Store) Recall(ctx context.Context, params RecallParams) ([]RecallResult, error) {
	if params.TopK <= 0 {
		params.TopK = s.cfg.DefaultTopK
	}
	if params.Threshold == 0 {
		params.Threshold = s.cfg.DefaultSimilarityThreshold
	}
	if params.VectorWeight == 0 {
		params.VectorWeight = s.cfg.HybridVectorWeight
	}
	if params.FTSWeight == 0 {
		params.FTSWeight = s.cfg.HybridFTSWeight
	}

	// Generate query embedding
	queryEmbedding, err := s.embed.Embed(params.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// If no namespaces provided, fallback to default
	if len(params.Namespaces) == 0 {
		params.Namespaces = []string{s.cfg.DefaultNamespace}
	}

	// Always include global namespace
	namespaces := append([]string{"global"}, params.Namespaces...)
	// Deduplicate
	seen := make(map[string]bool)
	uniqueNamespaces := []string{}
	for _, ns := range namespaces {
		if !seen[ns] {
			seen[ns] = true
			uniqueNamespaces = append(uniqueNamespaces, ns)
		}
	}

	// Run vector search and FTS in parallel
	vectorResults := make(chan map[uuid.UUID]float64, 1)
	ftsResults := make(chan map[uuid.UUID]float64, 1)
	errChan := make(chan error, 2)

	go func() {
		results, err := s.vectorSearch(ctx, queryEmbedding, uniqueNamespaces, params.TopK*2)
		if err != nil {
			errChan <- err
			return
		}
		vectorResults <- results
	}()

	go func() {
		results, err := s.fullTextSearch(ctx, params.Query, uniqueNamespaces, params.TopK*2)
		if err != nil {
			errChan <- err
			return
		}
		ftsResults <- results
	}()

	// Collect results
	var vecScores, ftsScores map[uuid.UUID]float64
	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			return nil, err
		case vecScores = <-vectorResults:
		case ftsScores = <-ftsResults:
		}
	}

	// RRF fusion with weights
	fusedScores := rrfFusion(vecScores, ftsScores, 60, params.VectorWeight, params.FTSWeight)

	// Sort by score
	type scoredMemory struct {
		id    uuid.UUID
		score float64
	}
	scored := make([]scoredMemory, 0, len(fusedScores))
	for id, score := range fusedScores {
		scored = append(scored, scoredMemory{id, score})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Apply threshold and get top-k
	var resultIDs []uuid.UUID
	for _, sm := range scored {
		if sm.score >= params.Threshold {
			resultIDs = append(resultIDs, sm.id)
		}
		if len(resultIDs) >= params.TopK {
			break
		}
	}

	if len(resultIDs) == 0 {
		return []RecallResult{}, nil
	}

	// Fetch full memory records
	memories, err := s.fetchMemoriesByIDs(ctx, resultIDs, params.Filter)
	if err != nil {
		return nil, err
	}

	// Build results with scores
	results := make([]RecallResult, 0, len(memories))
	for _, m := range memories {
		results = append(results, RecallResult{
			Memory: m,
			Score:  fusedScores[m.ID],
		})
	}

	return results, nil
}

func (s *Store) vectorSearch(ctx context.Context, embedding []float32, namespaces []string, limit int) (map[uuid.UUID]float64, error) {
	query := `
		SELECT id, 1 - (embedding <=> $1) AS similarity
		FROM memories
		WHERE namespace = ANY($2)
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	vec := pgvector.NewVector(embedding)
	rows, err := s.db.Pool().Query(ctx, query, vec, namespaces, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer rows.Close()

	results := make(map[uuid.UUID]float64)
	for rows.Next() {
		var id uuid.UUID
		var similarity float64
		if err := rows.Scan(&id, &similarity); err != nil {
			return nil, err
		}
		results[id] = similarity
	}

	return results, rows.Err()
}

func (s *Store) fullTextSearch(ctx context.Context, query string, namespaces []string, limit int) (map[uuid.UUID]float64, error) {
	// Clean and split the query into words
	words := strings.Fields(query)
	if len(words) == 0 {
		return make(map[uuid.UUID]float64), nil
	}

	// Join with OR operator and add prefix matching
	// e.g., "dog pet animal" -> "dog:* | pet:* | animal:*"
	for i, word := range words {
		// Basic sanitization to prevent tsquery syntax errors
		word = strings.ReplaceAll(word, "'", "''")
		word = strings.ReplaceAll(word, "\\", "")
		words[i] = word + ":*"
	}
	tsquery := strings.Join(words, " | ")

	sql := `
		SELECT id, ts_rank(search_vec, to_tsquery('english', $1)) AS rank
		FROM memories
		WHERE search_vec @@ to_tsquery('english', $1)
		  AND namespace = ANY($2)
		ORDER BY rank DESC
		LIMIT $3
	`

	rows, err := s.db.Pool().Query(ctx, sql, tsquery, namespaces, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search failed: %w", err)
	}
	defer rows.Close()

	results := make(map[uuid.UUID]float64)
	for rows.Next() {
		var id uuid.UUID
		var rank float64
		if err := rows.Scan(&id, &rank); err != nil {
			return nil, err
		}
		results[id] = rank
	}

	return results, rows.Err()
}

func rrfFusion(vectorScores, ftsScores map[uuid.UUID]float64, k int, vecWeight, ftsWeight float64) map[uuid.UUID]float64 {
	if vecWeight == 0 && ftsWeight == 0 {
		vecWeight = 0.7
		ftsWeight = 0.3
	}

	fused := make(map[uuid.UUID]float64)

	vecRanks := buildRanks(vectorScores)
	ftsRanks := buildRanks(ftsScores)

	allIDs := make(map[uuid.UUID]bool)
	for id := range vectorScores {
		allIDs[id] = true
	}
	for id := range ftsScores {
		allIDs[id] = true
	}

	for id := range allIDs {
		score := 0.0
		if rank, ok := vecRanks[id]; ok {
			score += vecWeight * (1.0 / (float64(k) + float64(rank)))
		}
		if rank, ok := ftsRanks[id]; ok {
			score += ftsWeight * (1.0 / (float64(k) + float64(rank)))
		}
		fused[id] = score
	}

	return fused
}

func buildRanks(scores map[uuid.UUID]float64) map[uuid.UUID]int {
	type item struct {
		id    uuid.UUID
		score float64
	}

	items := make([]item, 0, len(scores))
	for id, score := range scores {
		items = append(items, item{id, score})
	}

	// Sort by score descending
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	ranks := make(map[uuid.UUID]int)
	for i, item := range items {
		ranks[item.id] = i + 1 // 1-based rank
	}

	return ranks
}

func (s *Store) fetchMemoriesByIDs(ctx context.Context, ids []uuid.UUID, filter Filter) ([]Memory, error) {
	if len(ids) == 0 {
		return []Memory{}, nil
	}

	query := `
		SELECT id, namespace, content, content_hash, metadata, ttl_seconds, expires_at, created_at, updated_at
		FROM memories
		WHERE id = ANY($1)
		  AND (expires_at IS NULL OR expires_at > NOW())
	`
	args := []interface{}{ids}
	argIdx := 2

	if filter.Since != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filter.Since)
		argIdx++
	}

	if filter.Until != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filter.Until)
		argIdx++
	}

	if len(filter.Tags) > 0 {
		tagStrs := make([]string, len(filter.Tags))
		for i, t := range filter.Tags {
			tagStrs[i] = `"` + strings.ReplaceAll(t, `"`, `\"`) + `"`
		}
		query += fmt.Sprintf(" AND metadata->'tags' @> $%d::jsonb", argIdx)
		args = append(args, "["+strings.Join(tagStrs, ",")+"]")
		argIdx++
	}

	if filter.Source != "" {
		query += fmt.Sprintf(" AND metadata->>'source' = $%d", argIdx)
		args = append(args, filter.Source)
	}

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch memories: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}
