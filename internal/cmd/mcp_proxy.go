package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPProxy struct {
	serverURL string
	apiKey    string
	client    *http.Client
}

func NewMCPProxy(serverURL, apiKey string) *MCPProxy {
	return &MCPProxy{
		serverURL: serverURL,
		apiKey:    apiKey,
		client:    &http.Client{},
	}
}

func (p *MCPProxy) Run(ctx context.Context) error {
	tools, err := p.fetchTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch tools from server: %w", err)
	}

	server := mcp.NewServer(
		&mcp.Implementation{Name: "trindex", Version: "1.0.0"},
		nil,
	)

	for _, tool := range tools {
		toolCopy := tool
		server.AddTool(&toolCopy,
			func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return p.callTool(ctx, request)
			},
		)
	}

	return server.Run(ctx, &mcp.StdioTransport{})
}

func (p *MCPProxy) fetchTools(ctx context.Context) ([]mcp.Tool, error) {
	url := p.serverURL + "/api/mcp/tools"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if p.apiKey != "" {
		req.Header.Set("X-API-Key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tools []mcp.Tool `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Tools, nil
}

func (p *MCPProxy) callTool(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := p.serverURL + "/api/mcp/call"

	reqBody := map[string]interface{}{
		"name":      request.Params.Name,
		"arguments": request.Params.Arguments,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("X-API-Key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var mcpResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := &mcp.CallToolResult{
		Content: []mcp.Content{},
	}

	for _, content := range mcpResp.Content {
		if content.Type == "text" {
			result.Content = append(result.Content, &mcp.TextContent{
				Text: content.Text,
			})
		}
	}

	if mcpResp.IsError {
		result.IsError = true
	}

	return result, nil
}
