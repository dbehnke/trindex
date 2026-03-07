package mcp

import (
	"context"
	"time"

	"github.com/dbehnke/trindex/internal/memory"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// rememberInput represents the input for the remember tool
type rememberInput struct {
	Content   string                 `json:"content" jsonschema:"The memory text to store"`
	Namespace string                 `json:"namespace,omitempty" jsonschema:"Scope for this memory. If unsure, ALWAYS use 'default'."`
	Metadata  map[string]interface{} `json:"metadata,omitempty" jsonschema:"Arbitrary key/value tags"`
}

func (s *Server) registerRemember() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "remember",
		Description: "Store a memory with optional namespace and metadata",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in rememberInput) (*mcp.CallToolResult, any, error) {
		if in.Content == "" {
			return &mcp.CallToolResult{
				Content: errorResult("INVALID_INPUT", "content is required"),
			}, nil, nil
		}

		mem, err := s.store.Create(ctx, in.Content, in.Namespace, in.Metadata)
		if err != nil {
			return &mcp.CallToolResult{
				Content: errorResult("EMBED_FAILED", err.Error()),
			}, nil, nil
		}

		result := map[string]interface{}{
			"id":         mem.ID,
			"namespace":  mem.Namespace,
			"metadata":   mem.Metadata,
			"created_at": mem.CreatedAt,
		}

		return &mcp.CallToolResult{Content: successResult(result)}, nil, nil
	})
}

// recallInput represents the input for the recall tool
type recallInput struct {
	Query      string   `json:"query" jsonschema:"Natural language search query"`
	Namespaces []string `json:"namespaces,omitempty" jsonschema:"Namespaces to search. If unsure, ALWAYS use ['default']."`
	TopK       int      `json:"top_k,omitempty" jsonschema:"Number of results to return"`
	Threshold  float64  `json:"threshold,omitempty" jsonschema:"Minimum RRF similarity score (e.g. 0.0001 to 0.02). Default: 0.0001"`
	Filter     struct {
		Since  *time.Time `json:"since,omitempty" jsonschema:"Filter by start date"`
		Until  *time.Time `json:"until,omitempty" jsonschema:"Filter by end date"`
		Tags   []string   `json:"tags,omitempty" jsonschema:"Match any tag in metadata.tags"`
		Source string     `json:"source,omitempty" jsonschema:"Match metadata.source"`
	} `json:"filter,omitempty"`
}

func (s *Server) registerRecall() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "recall",
		Description: "Retrieve memories by semantic similarity. Use this tool PROACTIVELY whenever the user asks about past interactions, personal details, or facts you do not immediately know.",
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

		params := memory.RecallParams{
			Query:      in.Query,
			Namespaces: in.Namespaces,
			TopK:       in.TopK,
			Threshold:  in.Threshold,
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

		result := map[string]interface{}{
			"results":             results,
			"total":               len(results),
			"namespaces_searched": append([]string{"global"}, in.Namespaces...),
		}

		return &mcp.CallToolResult{Content: successResult(result)}, nil, nil
	})
}

// forgetInput represents the input for the forget tool
type forgetInput struct {
	ID        string `json:"id,omitempty" jsonschema:"Delete single memory by UUID"`
	Namespace string `json:"namespace,omitempty" jsonschema:"Delete all memories in namespace. If unsure, use 'default'."`
	Filter    struct {
		Before *time.Time `json:"before,omitempty" jsonschema:"Delete memories older than this"`
		Tags   []string   `json:"tags,omitempty" jsonschema:"Delete memories matching these tags"`
	} `json:"filter,omitempty"`
}

func (s *Server) registerForget() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "forget",
		Description: "Delete one or more memories. At least one of id, namespace, or filter must be provided.",
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
	Namespace string `json:"namespace,omitempty" jsonschema:"Filter by namespace. If unsure, use 'default'."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Number of results to return"`
	Offset    int    `json:"offset,omitempty" jsonschema:"Offset for pagination"`
	Order     string `json:"order,omitempty" jsonschema:"Sort order (asc or desc)"`
}

func (s *Server) registerList() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list",
		Description: "Browse memories without a semantic query",
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
	Namespace string `json:"namespace,omitempty" jsonschema:"Scope stats to namespace. If unsure, use 'default'."`
}

func (s *Server) registerStats() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "stats",
		Description: "Return memory statistics",
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
