package memory

import (
	"context"
	"fmt"
	"time"
)

// GetStats returns statistics about stored memories
func (s *Store) GetStats(ctx context.Context, namespace string) (*Stats, error) {
	stats := &Stats{
		ByNamespace: make(map[string]int64),
	}

	// Total count (with optional namespace filter)
	if err := s.getTotalCount(ctx, namespace, stats); err != nil {
		return nil, err
	}

	// Count by namespace
	if err := s.getCountByNamespace(ctx, stats); err != nil {
		return nil, err
	}

	// Recent 24h count
	if err := s.getRecentCount(ctx, namespace, stats); err != nil {
		return nil, err
	}

	// Oldest and newest
	if err := s.getTimeRange(ctx, namespace, stats); err != nil {
		return nil, err
	}

	// Top tags
	if err := s.getTopTags(ctx, namespace, stats); err != nil {
		return nil, err
	}

	// Config info
	stats.EmbeddingModel = s.embed.Model()
	stats.EmbedDimensions = s.embed.Dimensions()

	return stats, nil
}

func (s *Store) getTotalCount(ctx context.Context, namespace string, stats *Stats) error {
	query := `SELECT COUNT(*) FROM memories`
	var args []interface{}

	if namespace != "" {
		query += ` WHERE namespace = $1`
		args = append(args, namespace)
	}

	return s.db.Pool().QueryRow(ctx, query, args...).Scan(&stats.TotalMemories)
}

func (s *Store) getCountByNamespace(ctx context.Context, stats *Stats) error {
	rows, err := s.db.Pool().Query(ctx, `
		SELECT namespace, COUNT(*) 
		FROM memories 
		GROUP BY namespace`)
	if err != nil {
		return fmt.Errorf("failed to get namespace counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ns string
		var count int64
		if err := rows.Scan(&ns, &count); err != nil {
			return err
		}
		stats.ByNamespace[ns] = count
	}

	return rows.Err()
}

func (s *Store) getRecentCount(ctx context.Context, namespace string, stats *Stats) error {
	query := `SELECT COUNT(*) FROM memories WHERE created_at > NOW() - INTERVAL '24 hours'`
	var args []interface{}

	if namespace != "" {
		query += ` AND namespace = $1`
		args = append(args, namespace)
	}

	return s.db.Pool().QueryRow(ctx, query, args...).Scan(&stats.Recent24h)
}

func (s *Store) getTimeRange(ctx context.Context, namespace string, stats *Stats) error {
	query := `SELECT MIN(created_at), MAX(created_at) FROM memories`
	var args []interface{}

	if namespace != "" {
		query += ` WHERE namespace = $1`
		args = append(args, namespace)
	}

	var oldest, newest *time.Time
	if err := s.db.Pool().QueryRow(ctx, query, args...).Scan(&oldest, &newest); err != nil {
		return fmt.Errorf("failed to get time range: %w", err)
	}

	stats.OldestMemory = oldest
	stats.NewestMemory = newest

	return nil
}

func (s *Store) getTopTags(ctx context.Context, namespace string, stats *Stats) error {
	query := `
		SELECT jsonb_array_elements_text(metadata->'tags') AS tag, COUNT(*) AS cnt
		FROM memories
		WHERE metadata->'tags' IS NOT NULL
	`
	var args []interface{}

	if namespace != "" {
		query += ` AND namespace = $1`
		args = append(args, namespace)
	}

	query += `
		GROUP BY tag
		ORDER BY cnt DESC
		LIMIT 10
	`

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to get top tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		var count int64
		if err := rows.Scan(&tag, &count); err != nil {
			return err
		}
		stats.TopTags = append(stats.TopTags, tag)
	}

	return rows.Err()
}
