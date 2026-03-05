# MCP Client/Server Architecture Plan

## Overview

Redesign Trindex to use a client/server model where:
- **Client (`trindex mcp`)**: Thin MCP proxy with no local dependencies
- **Server (`trindex server`)**: Full stack with Postgres + Ollama

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         CLIENT (AI Agent Machine)                       │
│  ┌─────────────────┐      ┌──────────────────┐                         │
│  │   AI Agent      │stdio │  Trindex MCP     │ HTTP/JSON              │
│  │  (opencode)     │──────▶│  PROXY CLIENT    │─────────────────────┐  │
│  │                 │      │  - No DB         │                     │  │
│  │                 │      │  - No Ollama     │                     │  │
│  │                 │      │  - Just config:  │                     │  │
│  │                 │      │    TRINDEX_URL   │                     │  │
│  │                 │      │    TRINDEX_KEY   │                     │  │
│  └─────────────────┘      └──────────────────┘                     │  │
└──────────────────────────────────────────────────────────────────────┼──┘
                                                                       │
                              Network (HTTP/JSON)                      │
                                                                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         SERVER (Can be anywhere)                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                     Trindex Server                               │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────────┐ │  │
│  │  │ HTTP Server │  │ MCP Bridge  │  │   Web Dashboard          │ │  │
│  │  │  :9636      │  │  (optional) │  │   (Vue + Tailwind)       │ │  │
│  │  └─────────────┘  └─────────────┘  └──────────────────────────┘ │  │
│  │         │                │                                      │  │
│  │         └────────────────┘                                      │  │
│  │                   │                                             │  │
│  │                   ▼                                             │  │
│  │  ┌──────────────────────────────────────────────────────────┐  │  │
│  │  │                 Business Logic                           │  │  │
│  │  │  - Memory Store (recall, remember, forget, list)         │  │  │
│  │  │  - Stats                                               │  │  │
│  │  │  - Import/Export                                       │  │  │
│  │  └──────────────────────────────────────────────────────────┘  │  │
│  │                   │                                             │  │
│  │         ┌─────────┴──────────┐                                  │  │
│  │         ▼                    ▼                                  │  │
│  │  ┌──────────────┐                                             │  │
│  │  │  PostgreSQL  │                                             │  │
│  │  │  (pgvector)  │                                             │  │
│  │  └──────────────┘                                             │  │
│  │                                                               │  │
│  │  Connects to Ollama on host via host.docker.internal:11434   │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

## Component Changes

### 1. Client (`trindex mcp`)

**Before:**
- Required DATABASE_URL
- Required EMBED_BASE_URL
- Ran full MCP server with local DB

**After:**
- No local dependencies
- Only needs:
  ```
  TRINDEX_URL=http://server:9636
  TRINDEX_API_KEY=secret
  ```
- Runs MCP stdio → forwards to server HTTP

**Implementation:**
```go
// internal/mcp/client/proxy.go
func RunProxy(ctx context.Context, serverURL, apiKey string) error {
    // Create MCP server that handles stdio
    mcpServer := server.NewMCPServer()
    
    // Override tool handlers to proxy to HTTP
    mcpServer.RegisterTool("remember", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Convert MCP request to HTTP POST /api/mcp/remember
        return proxyToServer(serverURL, apiKey, "remember", req)
    })
    
    // ... other tools
    
    return mcpServer.RunStdio(ctx)
}
```

### 2. Server (`trindex server`)

**Before:**
- HTTP REST API only
- Separate from MCP

**After:**
- HTTP REST API (existing)
- **NEW**: MCP-over-HTTP endpoints
- Unified on port 9636

**New Endpoints:**
```
POST /api/mcp/remember     - MCP remember tool
POST /api/mcp/recall       - MCP recall tool
POST /api/mcp/forget       - MCP forget tool
POST /api/mcp/list         - MCP list tool
POST /api/mcp/stats        - MCP stats tool
GET  /api/mcp/tools        - List available tools (MCP discovery)
```

**Implementation:**
```go
// internal/web/server.go
func (s *Server) setupRoutes() {
    // Existing REST API
    r.Get("/api/memories", s.listMemories)
    r.Post("/api/memories", s.createMemory)
    // ... etc
    
    // NEW: MCP-over-HTTP endpoints
    r.Get("/api/mcp/tools", s.mcpTools)
    r.Post("/api/mcp/remember", s.mcpRemember)
    r.Post("/api/mcp/recall", s.mcpRecall)
    r.Post("/api/mcp/forget", s.mcpForget)
    r.Post("/api/mcp/list", s.mcpList)
    r.Post("/api/mcp/stats", s.mcpStats)
}
```

### 3. Configuration Split

**Client Config** (`~/.config/trindex/client.yaml`):
```yaml
# Client-only configuration
server_url: "http://localhost:9636"
api_key: "your-secret-key"

# Optional: for multiple servers
profiles:
  local:
    url: "http://localhost:9636"
    key: "dev-key"
  production:
    url: "https://brain.example.com"
    key: "prod-key"
```

**Server Config** (`docker-compose.yml` or `~/.config/trindex/server.yaml`):
```yaml
# Server configuration
http_host: "0.0.0.0"
http_port: "9636"
api_key: "your-secret-key"  # Required for security

# Database
database_url: "postgres://trindex:trindex@postgres:5432/trindex?sslmode=disable"

# Embedding (Ollama on host)
embed_base_url: "http://host.docker.internal:11434/v1"
embed_model: "nomic-embed-text"
embed_api_key: "ollama"
embed_dimensions: 768

# Other settings...
```

### 4. Docker Compose (Server Only)

**Prerequisite:** Ollama must be installed and running on the host (not in Docker):
```bash
# macOS
brew install ollama
ollama serve
ollama pull nomic-embed-text

# Linux
curl -fsSL https://ollama.com/install.sh | sh
ollama serve
ollama pull nomic-embed-text
```

```yaml
version: "3.8"

services:
  postgres:
    image: pgvector/pgvector:pg17
    environment:
      POSTGRES_USER: trindex
      POSTGRES_PASSWORD: trindex
      POSTGRES_DB: trindex
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U trindex"]
      interval: 5s
      timeout: 5s
      retries: 5

  trindex-server:
    build: .
    ports:
      - "9636:9636"
    environment:
      # HTTP Server
      HTTP_HOST: "0.0.0.0"
      HTTP_PORT: "9636"
      TRINDEX_API_KEY: ${TRINDEX_API_KEY:-change-me-in-production}
      
      # Database
      DATABASE_URL: postgres://trindex:trindex@postgres:5432/trindex?sslmode=disable
      
      # Embedding via Ollama on host (not in Docker)
      EMBED_BASE_URL: http://host.docker.internal:11434/v1
      EMBED_MODEL: nomic-embed-text
      EMBED_API_KEY: ollama
      EMBED_DIMENSIONS: 768
    depends_on:
      postgres:
        condition: service_healthy
    extra_hosts:
      - "host.docker.internal:host-gateway"
    command: ["server"]

volumes:
  pgdata:
```

**Why Ollama on the host?**
- GPU acceleration works natively (no Docker GPU passthrough complexity)
- Better performance (no container overhead)
- Ollama can use host GPU drivers directly
- Simpler deployment for users

### 5. Client Usage Examples

**With explicit config:**
```bash
# Set env vars
export TRINDEX_URL=http://localhost:9636
export TRINDEX_API_KEY=my-secret

# Run MCP client (no local dependencies!)
trindex mcp
```

**With config file:**
```bash
# ~/.config/trindex/config.yaml
server_url: "http://brain.example.com:9636"
api_key: "production-key"

# Just run
trindex mcp
```

**With flags:**
```bash
trindex mcp --server http://localhost:9636 --api-key secret
```

### 6. Server Usage

```bash
# Docker (recommended)
docker compose up -d

# Or binary (requires Postgres + Ollama running)
export DATABASE_URL=postgres://...
export EMBED_BASE_URL=http://localhost:11434/v1
trindex server
```

## Migration Path

### Phase 1: Implement MCP-over-HTTP on Server
1. Add `/api/mcp/*` endpoints to existing server
2. Convert MCP tool requests to internal calls
3. Return MCP-compatible responses

### Phase 2: Create MCP Proxy Client
1. New `trindex mcp` implementation
2. Remove DB/embedding dependencies
3. HTTP client to talk to server
4. Config for server URL + API key

### Phase 3: Update Docker Compose
1. Add Ollama service
2. Update environment variables
3. Document server-only deployment

### Phase 4: Documentation & Examples
1. Update README with new architecture
2. Create client setup guide
3. Create server deployment guide
4. Update MCP configuration examples

## Benefits

1. **Zero client setup**: Just download binary, set URL, run
2. **Centralized brain**: Multiple agents share one database
3. **Resource efficiency**: One Ollama instance, not per-agent
4. **Simplified deployment**: Server can run anywhere (cloud, NAS, etc.)
5. **Better security**: API keys instead of DB credentials on clients
6. **Easy scaling**: Server can be scaled independently

## Security Considerations

1. **API Key required**: Server must have `TRINDEX_API_KEY` set
2. **HTTPS in production**: Clients should use `https://`
3. **Network isolation**: Server should be on private network or VPN
4. **Rate limiting**: Consider adding rate limits per API key

## Open Questions

1. Should we support multiple API keys (one per agent)?
2. Do we need client-side caching?
3. Should MCP tools be dynamically discovered from server?
4. How to handle server upgrades while clients connected?
5. Should we add a `trindex doctor --remote` to test client connectivity?

## Implementation Priority

1. **P0**: MCP-over-HTTP endpoints on server
2. **P0**: Refactor `trindex mcp` to proxy client
3. **P1**: Update docker-compose with Ollama
4. **P1**: Config file split (client vs server)
5. **P2**: Documentation updates
6. **P2**: Add `trindex client doctor` command
