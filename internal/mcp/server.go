package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/memory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server and dependencies
type Server struct {
	cfg    *config.Config
	db     *db.DB
	embed  *embed.Client
	store  *memory.Store
	server *mcp.Server
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config, database *db.DB, embedClient *embed.Client) *Server {
	store := memory.NewStore(database, embedClient, cfg)

	return &Server{
		cfg:    cfg,
		db:     database,
		embed:  embedClient,
		store:  store,
		server: mcp.NewServer(&mcp.Implementation{Name: "trindex", Version: "1.0.0"}, nil),
	}
}

// RegisterTools registers all MCP tools
func (s *Server) RegisterTools() {
	s.registerRemember()
	s.registerRecall()
	s.registerForget()
	s.registerList()
	s.registerStats()
}

// Run starts the MCP server with stdio transport
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}

// errorResult creates a JSON error result
func errorResult(code, message string) []mcp.Content {
	result := map[string]string{
		"error":   code,
		"message": message,
	}
	jsonBytes, _ := json.Marshal(result)
	return []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}}
}

// successResult creates a JSON success result
func successResult(data interface{}) []mcp.Content {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return errorResult("SERIALIZE_ERROR", fmt.Sprintf("failed to serialize result: %v", err))
	}
	return []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}}
}
