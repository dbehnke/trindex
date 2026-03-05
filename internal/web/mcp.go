package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dbehnke/trindex/internal/memory"
	"github.com/google/uuid"
)

// MCPRequest represents an MCP tool call request
type MCPRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// MCPResponse represents an MCP tool call response
type MCPResponse struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent represents content in an MCP response
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// handleMCPTools returns the list of available MCP tools
func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	tools := []map[string]interface{}{
		{
			"name":        "remember",
			"description": "Store a memory with optional namespace and metadata. Returns the created memory ID.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The memory text to store",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Scope for this memory (default: 'default')",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Arbitrary key/value tags",
					},
				},
				"required": []string{"content"},
			},
		},
		{
			"name":        "recall",
			"description": "Retrieve memories by semantic similarity using hybrid search. Always includes 'global' namespace.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Natural language search query",
					},
					"namespaces": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Namespaces to search (global always included)",
					},
					"top_k": map[string]interface{}{
						"type":        "integer",
						"description": "Number of results to return",
						"default":     10,
					},
					"threshold": map[string]interface{}{
						"type":        "number",
						"description": "Minimum similarity score (0.0-1.0)",
						"default":     0.7,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "forget",
			"description": "Delete one or more memories by ID, namespace, or filter.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "Delete single memory by UUID",
					},
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Delete all memories in namespace",
					},
				},
			},
		},
		{
			"name":        "list",
			"description": "Browse memories without semantic query. Useful for inspection.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Filter by namespace",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results",
						"default":     20,
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Pagination offset",
						"default":     0,
					},
				},
			},
		},
		{
			"name":        "stats",
			"description": "Return memory statistics. Useful for monitoring.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Scope stats to namespace (omit for global)",
					},
				},
			},
		},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tools": tools,
	})
}

// handleMCPCall handles MCP tool calls via HTTP
func (s *Server) handleMCPCall(w http.ResponseWriter, r *http.Request) {
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondMCPError(w, "failed to parse request: "+err.Error())
		return
	}

	switch req.Name {
	case "remember":
		s.handleMCPRemember(w, r, req.Arguments)
	case "recall":
		s.handleMCPRecall(w, r, req.Arguments)
	case "forget":
		s.handleMCPForget(w, r, req.Arguments)
	case "list":
		s.handleMCPList(w, r, req.Arguments)
	case "stats":
		s.handleMCPStats(w, r, req.Arguments)
	default:
		respondMCPError(w, "unknown tool: "+req.Name)
	}
}

func (s *Server) handleMCPRemember(w http.ResponseWriter, r *http.Request, args json.RawMessage) {
	var params struct {
		Content   string                 `json:"content"`
		Namespace string                 `json:"namespace"`
		Metadata  map[string]interface{} `json:"metadata"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		respondMCPError(w, "invalid arguments: "+err.Error())
		return
	}

	if params.Content == "" {
		respondMCPError(w, "content is required")
		return
	}

	mem, err := s.store.Create(r.Context(), params.Content, params.Namespace, params.Metadata)
	if err != nil {
		respondMCPError(w, "failed to create memory: "+err.Error())
		return
	}

	result := fmt.Sprintf("Created memory: %s\nNamespace: %s\nContent: %s",
		mem.ID.String(), mem.Namespace, mem.Content)

	respondMCP(w, result)
}

func (s *Server) handleMCPRecall(w http.ResponseWriter, r *http.Request, args json.RawMessage) {
	var params struct {
		Query      string   `json:"query"`
		Namespaces []string `json:"namespaces"`
		TopK       int      `json:"top_k"`
		Threshold  float64  `json:"threshold"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		respondMCPError(w, "invalid arguments: "+err.Error())
		return
	}

	if params.Query == "" {
		respondMCPError(w, "query is required")
		return
	}

	if params.TopK == 0 {
		params.TopK = s.cfg.DefaultTopK
	}
	if params.Threshold == 0 {
		params.Threshold = s.cfg.DefaultSimilarityThreshold
	}

	// Always include global namespace
	if len(params.Namespaces) == 0 {
		params.Namespaces = []string{"global"}
	} else {
		// Check if global is already included
		hasGlobal := false
		for _, ns := range params.Namespaces {
			if ns == "global" {
				hasGlobal = true
				break
			}
		}
		if !hasGlobal {
			params.Namespaces = append(params.Namespaces, "global")
		}
	}

	recallParams := memory.RecallParams{
		Query:      params.Query,
		Namespaces: params.Namespaces,
		TopK:       params.TopK,
		Threshold:  params.Threshold,
	}

	results, err := s.store.Recall(r.Context(), recallParams)
	if err != nil {
		respondMCPError(w, "search failed: "+err.Error())
		return
	}

	if len(results) == 0 {
		respondMCP(w, "No memories found matching your query.")
		return
	}

	var output string
	output += fmt.Sprintf("Found %d memories:\n\n", len(results))
	for i, result := range results {
		output += fmt.Sprintf("%d. [%.2f] %s\n   ID: %s\n   Namespace: %s\n   Content: %s\n\n",
			i+1, result.Score, result.Content[:min(len(result.Content), 100)],
			result.ID.String(), result.Namespace, result.Content)
	}

	respondMCP(w, output)
}

func (s *Server) handleMCPForget(w http.ResponseWriter, r *http.Request, args json.RawMessage) {
	var params struct {
		ID        string `json:"id"`
		Namespace string `json:"namespace"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		respondMCPError(w, "invalid arguments: "+err.Error())
		return
	}

	if params.ID == "" && params.Namespace == "" {
		respondMCPError(w, "either id or namespace is required")
		return
	}

	if params.ID != "" {
		id, err := uuid.Parse(params.ID)
		if err != nil {
			respondMCPError(w, "invalid memory ID: "+err.Error())
			return
		}

		if err := s.store.DeleteByID(r.Context(), id); err != nil {
			respondMCPError(w, "failed to delete memory: "+err.Error())
			return
		}

		respondMCP(w, "Memory deleted successfully: "+params.ID)
		return
	}

	count, err := s.store.DeleteByNamespace(r.Context(), params.Namespace, memory.ForgetFilter{})
	if err != nil {
		respondMCPError(w, "failed to delete memories: "+err.Error())
		return
	}

	respondMCP(w, fmt.Sprintf("Deleted %d memories from namespace: %s", count, params.Namespace))
}

func (s *Server) handleMCPList(w http.ResponseWriter, r *http.Request, args json.RawMessage) {
	var params struct {
		Namespace string `json:"namespace"`
		Limit     int    `json:"limit"`
		Offset    int    `json:"offset"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		respondMCPError(w, "invalid arguments: "+err.Error())
		return
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	listParams := memory.ListParams{
		Namespace: params.Namespace,
		Limit:     params.Limit,
		Offset:    params.Offset,
		Order:     "desc",
	}

	memories, err := s.store.List(r.Context(), listParams)
	if err != nil {
		respondMCPError(w, "failed to list memories: "+err.Error())
		return
	}

	if len(memories) == 0 {
		respondMCP(w, "No memories found.")
		return
	}

	var output string
	output += fmt.Sprintf("Found %d memories:\n\n", len(memories))
	for i, mem := range memories {
		content := mem.Content
		if len(content) > 100 {
			content = content[:97] + "..."
		}
		output += fmt.Sprintf("%d. %s | %s | %s\n",
			i+1, mem.ID.String()[:8], mem.Namespace, content)
	}

	respondMCP(w, output)
}

func (s *Server) handleMCPStats(w http.ResponseWriter, r *http.Request, args json.RawMessage) {
	var params struct {
		Namespace string `json:"namespace"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		respondMCPError(w, "invalid arguments: "+err.Error())
		return
	}

	stats, err := s.store.GetStats(r.Context(), params.Namespace)
	if err != nil {
		respondMCPError(w, "failed to get stats: "+err.Error())
		return
	}

	var output string
	output += "📊 Trindex Statistics\n\n"
	output += fmt.Sprintf("Total Memories: %d\n", stats.TotalMemories)
	output += fmt.Sprintf("Recent (24h): %d\n", stats.Recent24h)
	output += fmt.Sprintf("Embedding Model: %s\n", stats.EmbeddingModel)
	output += fmt.Sprintf("Dimensions: %d\n", stats.EmbedDimensions)

	if len(stats.ByNamespace) > 0 {
		output += "\nBy Namespace:\n"
		for ns, count := range stats.ByNamespace {
			output += fmt.Sprintf("  %s: %d\n", ns, count)
		}
	}

	respondMCP(w, output)
}

func respondMCP(w http.ResponseWriter, text string) {
	resp := MCPResponse{
		Content: []MCPContent{
			{Type: "text", Text: text},
		},
	}
	respondJSON(w, http.StatusOK, resp)
}

func respondMCPError(w http.ResponseWriter, message string) {
	resp := MCPResponse{
		Content: []MCPContent{
			{Type: "text", Text: message},
		},
		IsError: true,
	}
	respondJSON(w, http.StatusOK, resp)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
