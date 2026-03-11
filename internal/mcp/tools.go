package mcp

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/dbehnke/trindex/internal/memory"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type rememberInput struct {
	Content                string                 `json:"content" jsonschema:"The memory text to store. Write 1-3 concise sentences that state the fact directly. Good candidates: decisions made, user preferences, patterns discovered, task outcomes, architectural choices, bug root causes. Avoid trivial or ephemeral facts."`
	Namespace              string                 `json:"namespace,omitempty" jsonschema:"Scope for this memory. Use 'global' for cross-agent user facts (preferences, identity, persistent context). Use a project namespace (e.g. 'trindex', 'myapp') for task-specific memories. Use 'default' when unsure. Follow hierarchical convention: global > project:{name} > agent:{name} > session:{id}."`
	Metadata               map[string]interface{} `json:"metadata,omitempty" jsonschema:"Arbitrary key/value tags. Recommended fields: type (e.g. 'decision', 'preference', 'pattern', 'bug', 'outcome'), tags ([]string), agent (agent name), project (project name), source (e.g. 'session', 'user-statement')."`
	SkipDuplicateThreshold float64                `json:"skip_duplicate_threshold,omitempty" jsonschema:"Similarity threshold (0.0-1.0) for deduplication. If a memory with similarity >= threshold exists in the same namespace, skip storing. Use 0.95 for exact duplicates, 0.85 for semantic duplicates. Set to 0 to disable (default)."`
	TTLSeconds             int32                  `json:"ttl_seconds,omitempty" jsonschema:"Time-to-live in seconds. Memory will be automatically deleted after this duration. Use for ephemeral context like session IDs, temporary files, transient errors. session:* namespaces default to 86400 (24h)."`
}

func (s *Server) registerRemember() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "remember",
		Description: "Store a memory with optional namespace and metadata. Call this after making a significant decision, learning a user preference, identifying a recurring pattern, completing a task, or discovering a bug root cause. Write content as 1-3 concise sentences stating the fact directly. Use namespace 'global' for cross-agent user facts; use a project-specific namespace for task-specific memories; use 'default' when unsure. Include metadata fields like type, tags, agent, and project to make memories easier to filter later. Use skip_duplicate_threshold to avoid storing duplicates (0.95 for exact, 0.85 for semantic matches).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in rememberInput) (*mcp.CallToolResult, any, error) {
		if in.Content == "" {
			return &mcp.CallToolResult{
				Content: errorResult("INVALID_INPUT", "content is required"),
			}, nil, nil
		}

		namespace := in.Namespace
		if namespace == "" {
			namespace = s.cfg.DefaultNamespace
		}

		ttlSeconds := in.TTLSeconds
		if ttlSeconds == 0 && strings.HasPrefix(namespace, "session:") {
			ttlSeconds = 86400
		}

		if in.SkipDuplicateThreshold > 0 {
			duplicate, err := s.checkDuplicate(ctx, in.Content, namespace, in.SkipDuplicateThreshold)
			if err != nil {
				slog.Warn("deduplication check failed", "error", err)
			} else if duplicate != nil {
				return &mcp.CallToolResult{
					Content: successResult(map[string]interface{}{
						"status":      "duplicate_skipped",
						"existing_id": duplicate.ID,
						"namespace":   duplicate.Namespace,
						"similarity":  duplicate.Score,
					}),
				}, nil, nil
			}
		}

		mem, err := s.store.CreateWithParams(ctx, memory.CreateParams{
			Content:    in.Content,
			Namespace:  namespace,
			Metadata:   in.Metadata,
			TTLSeconds: ttlSeconds,
		})
		if err != nil {
			return &mcp.CallToolResult{
				Content: errorResult("EMBED_FAILED", err.Error()),
			}, nil, nil
		}

		result := map[string]interface{}{
			"id":           mem.ID,
			"namespace":    mem.Namespace,
			"metadata":     mem.Metadata,
			"created_at":   mem.CreatedAt,
			"expires_at":   mem.ExpiresAt,
			"content_hash": mem.ContentHash,
		}

		return &mcp.CallToolResult{Content: successResult(result)}, nil, nil
	})
}

func (s *Server) checkDuplicate(ctx context.Context, content, namespace string, threshold float64) (*memory.RecallResult, error) {
	params := memory.RecallParams{
		Query:      content,
		Namespaces: []string{namespace},
		TopK:       1,
		Threshold:  threshold,
	}

	results, err := s.store.Recall(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 && results[0].Score >= threshold {
		return &results[0], nil
	}

	return nil, nil
}

// recallInput represents the input for the recall tool
type recallInput struct {
	Query        string   `json:"query" jsonschema:"Natural language search query. Phrase it as a statement or question describing what you are looking for, e.g. 'user preferred database' or 'how was the authentication bug fixed'."`
	Namespaces   []string `json:"namespaces,omitempty" jsonschema:"Namespaces to search. 'global' is always searched automatically regardless of what you pass here. Use a project namespace (e.g. ['trindex']) for project-specific recall. Use ['default'] when unsure. You may pass multiple namespaces to cast a wide net."`
	TopK         int      `json:"top_k,omitempty" jsonschema:"Number of results to return. Default: 10."`
	Threshold    float64  `json:"threshold,omitempty" jsonschema:"Minimum RRF similarity score. Default 0.0001 casts a wide net. Raise to 0.005-0.02 for higher-precision retrieval when you only want close matches."`
	VectorWeight float64  `json:"vector_weight,omitempty" jsonschema:"Semantic search weight 0.0-1.0. Default 0 uses server config (0.7). Increase for conceptual or paraphrased queries where exact keywords may not match."`
	FTSWeight    float64  `json:"fts_weight,omitempty" jsonschema:"Full-text search weight 0.0-1.0. Default 0 uses server config (0.3). Increase for exact-term queries like function names, error codes, or identifiers."`
	Filter       struct {
		Since  *time.Time `json:"since,omitempty" jsonschema:"Filter by start date"`
		Until  *time.Time `json:"until,omitempty" jsonschema:"Filter by end date"`
		Tags   []string   `json:"tags,omitempty" jsonschema:"Match any tag in metadata.tags"`
		Source string     `json:"source,omitempty" jsonschema:"Match metadata.source"`
	} `json:"filter,omitempty"`
}

func (s *Server) registerRecall() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "recall",
		Description: "Retrieve memories by semantic similarity using hybrid search (vector + full-text, fused with RRF). Call this PROACTIVELY at session start to orient yourself, before making significant decisions, and whenever you encounter a pattern or bug that might have been seen before. The 'global' namespace is always searched automatically — you do not need to include it. Use threshold 0.0001 (default) for a wide net; raise to 0.005-0.02 for precision. Use vector_weight high (e.g. 0.9) for conceptual queries; use fts_weight high (e.g. 0.9) for exact term lookups. Pass multiple namespaces to search across project and personal context simultaneously.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in recallInput) (*mcp.CallToolResult, any, error) {
		if in.Query == "" {
			return &mcp.CallToolResult{
				Content: errorResult("INVALID_INPUT", "query is required"),
			}, nil, nil
		}

		if in.TopK <= 0 {
			in.TopK = s.cfg.DefaultTopK
		}
		if in.Threshold == 0 {
			in.Threshold = s.cfg.DefaultSimilarityThreshold
		}
		if len(in.Namespaces) == 0 {
			in.Namespaces = []string{s.cfg.DefaultNamespace}
		}

		params := memory.RecallParams{
			Query:        in.Query,
			Namespaces:   in.Namespaces,
			TopK:         in.TopK,
			Threshold:    in.Threshold,
			VectorWeight: in.VectorWeight,
			FTSWeight:    in.FTSWeight,
			Filter: memory.Filter{
				Since:  in.Filter.Since,
				Until:  in.Filter.Until,
				Tags:   in.Filter.Tags,
				Source: in.Filter.Source,
			},
		}

		results, err := s.store.Recall(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: errorResult("RECALL_FAILED", err.Error()),
			}, nil, nil
		}

		// Build namespaces_searched to match what recall.go actually searches:
		// always prepends "global", then deduplicates.
		allNs := append([]string{"global"}, in.Namespaces...)
		seen := make(map[string]bool)
		uniqueNs := []string{}
		for _, ns := range allNs {
			if !seen[ns] {
				seen[ns] = true
				uniqueNs = append(uniqueNs, ns)
			}
		}

		result := map[string]interface{}{
			"results":             results,
			"total":               len(results),
			"namespaces_searched": uniqueNs,
		}

		return &mcp.CallToolResult{Content: successResult(result)}, nil, nil
	})
}

// forgetInput represents the input for the forget tool
type forgetInput struct {
	ID        string `json:"id,omitempty" jsonschema:"Delete a single memory by its UUID. Use this for surgical, precise deletion when you know the exact memory ID from a previous recall or list result."`
	Namespace string `json:"namespace,omitempty" jsonschema:"Delete all memories in this namespace, optionally scoped by filter. Use for bulk pruning of a project or stale namespace. If unsure, use 'default'."`
	Filter    struct {
		Before *time.Time `json:"before,omitempty" jsonschema:"Delete memories older than this timestamp. Combine with namespace for scoped cleanup."`
		Tags   []string   `json:"tags,omitempty" jsonschema:"Delete memories that have any of these tags in metadata.tags. Combine with namespace for scoped cleanup."`
	} `json:"filter,omitempty"`
}

func (s *Server) registerForget() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "forget",
		Description: "Delete one or more memories. Use this when the user explicitly asks to forget something, when a memory is incorrect or stale, or to deduplicate redundant memories. Provide 'id' for surgical single-memory deletion (preferred when you have the UUID). Provide 'namespace' plus optional filter for bulk pruning. At least one of id, namespace, or filter must be provided — this tool will never delete without an explicit target.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in forgetInput) (*mcp.CallToolResult, any, error) {
		if in.ID == "" && in.Namespace == "" && in.Filter.Before == nil && len(in.Filter.Tags) == 0 {
			return &mcp.CallToolResult{
				Content: errorResult("NAMESPACE_REQUIRED", "at least one of id, namespace, or filter must be provided"),
			}, nil, nil
		}

		if in.ID != "" {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return &mcp.CallToolResult{
					Content: errorResult("INVALID_INPUT", "invalid UUID: "+in.ID),
				}, nil, nil
			}
			if err := s.store.DeleteByID(ctx, id); err != nil {
				return &mcp.CallToolResult{
					Content: errorResult("DELETE_FAILED", err.Error()),
				}, nil, nil
			}
			return &mcp.CallToolResult{Content: successResult(map[string]string{"status": "deleted"})}, nil, nil
		}

		filter := memory.ForgetFilter{
			Before: in.Filter.Before,
			Tags:   in.Filter.Tags,
		}
		count, err := s.store.DeleteByNamespace(ctx, in.Namespace, filter)
		if err != nil {
			return &mcp.CallToolResult{
				Content: errorResult("DELETE_FAILED", err.Error()),
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: successResult(map[string]interface{}{
				"status": "deleted",
				"count":  count,
			}),
		}, nil, nil
	})
}

// listInput represents the input for the list tool
type listInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Filter by namespace. Use this to inspect all memories in a specific namespace. If unsure, use 'default'."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Number of results to return. Default: 20."`
	Offset    int    `json:"offset,omitempty" jsonschema:"Offset for pagination."`
	Order     string `json:"order,omitempty" jsonschema:"Sort order: 'asc' or 'desc' by created_at. Use 'desc' (default) to see most recent memories first."`
}

func (s *Server) registerList() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list",
		Description: "Browse memories by recency without a semantic query. Prefer this over recall when you want to audit or inspect a namespace (e.g. 'what did I store in this project?'), find the most recent memories regardless of topic, or check whether a namespace is empty. Unlike recall, list does not perform semantic search and does not automatically include the global namespace — it returns exactly what is in the specified namespace ordered by time.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in listInput) (*mcp.CallToolResult, any, error) {
		params := memory.ListParams{
			Namespace: in.Namespace,
			Limit:     in.Limit,
			Offset:    in.Offset,
			Order:     in.Order,
		}

		memories, err := s.store.List(ctx, params)
		if err != nil {
			return &mcp.CallToolResult{
				Content: errorResult("LIST_FAILED", err.Error()),
			}, nil, nil
		}

		return &mcp.CallToolResult{
			Content: successResult(map[string]interface{}{
				"memories": memories,
				"total":    len(memories),
			}),
		}, nil, nil
	})
}

// statsInput represents the input for the stats tool
type statsInput struct {
	Namespace string `json:"namespace,omitempty" jsonschema:"Scope stats to a specific namespace. Omit to get global stats across all namespaces, which shows a full breakdown by namespace. If unsure, omit this field."`
}

func (s *Server) registerStats() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "stats",
		Description: "Return memory statistics including total count, breakdown by namespace, recent activity, and top tags. Call this at session start to orient yourself (see what namespaces exist and how much is stored). Also useful before an export or cleanup to scope the operation, and after a batch of remember calls to confirm they persisted. Omit namespace to get a full global overview; provide namespace to scope stats to a single namespace.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in statsInput) (*mcp.CallToolResult, any, error) {
		stats, err := s.store.GetStats(ctx, in.Namespace)
		if err != nil {
			return &mcp.CallToolResult{
				Content: errorResult("STATS_FAILED", err.Error()),
			}, nil, nil
		}

		return &mcp.CallToolResult{Content: successResult(stats)}, nil, nil
	})
}
