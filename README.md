# Trindex

Persistent semantic memory for AI agents via MCP (Model Context Protocol).

## Overview

Trindex is a standalone Go binary that provides persistent, semantic memory for AI agents. It stores memories as vectors in Postgres with pgvector, enabling semantic search and hybrid retrieval (vector + full-text search).

### Key Features

- **🔍 Hybrid Search** — Combines vector similarity (cosine) with full-text search (tsvector), fused with Reciprocal Rank Fusion (RRF)
- **🏷️ Hierarchical Namespaces** — Organized scoping: `global > project:{name} > agent:{name} > session:{id}`
- **🧠 Context Window Ranking** — Intelligent memory ranking for LLM prompts (relevance + recency + importance)
- **🛂 Context Passport** — Cross-system context transfer (Linear, GitHub, agent handoff)
- **🔄 Deduplication** — Client-side (threshold-based) and server-side (content hash) duplicate prevention
- **⏰ TTL Support** — Time-to-live for temporary memories with automatic cleanup
- **🌐 Multi-Agent** — Share one memory server across multiple AI agents (OpenCode, Claude Code, Cursor, etc.)
- **🖥️ Web UI** — Built-in Vue.js interface for browsing and managing memories
- **🔌 MCP Native** — Works with any MCP-compatible agent via stdio transport

## Quick Start

### ⚙️ Cognitive Evaluation & Stress Testing

Trindex includes a built-in, standalone CLI test suite designed to empirically prove Agent semantic recall precision under massive load.

Using `testcontainers-go`, the suite spins up an ephemeral, rootless `pgvector` container. The `generator` then probabilistically creates thousands of rows of multi-tenant "semantic noise" (dense tech ops jargon) and injects specific trackable targets deep within the noise. It then executes parallel Hybrid Search (`RRF`) queries to prove that the vector and full-text search thresholds can seamlessly recall the needle in the 10,000+ vector haystack with **100% precision**.

To run the agent-persona cognitive benchmark locally:

```bash
task eval
```

To run the massive generative cross-tenant stress test:

```bash
task eval -- -mode=stress -users=20 -noise=500
```

### Prerequisites

- Go 1.26+ (for building from source)
- Docker and Docker Compose (recommended)
- Postgres 17 with pgvector extension
- Ollama (for embeddings) - must run on the host for GPU acceleration

### Installation

#### Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/dbehnke/trindex.git
cd trindex

# Install and start Ollama (on the host, not in Docker)
# macOS:
brew install ollama && ollama serve
# Linux:
curl -fsSL https://ollama.com/install.sh | sh && ollama serve

# Pull the embedding model (one-time)
ollama pull nomic-embed-text

# Copy environment file
cp .env.example .env

# Start Trindex server (Postgres + Trindex)
docker compose up -d

# Verify it's working
./trindex doctor
```

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/dbehnke/trindex.git
cd trindex

# Build the binary
go build -o trindex ./cmd/trindex

# Run diagnostics to verify configuration
./trindex doctor
```

## CLI Commands

Trindex provides a command-line interface with explicit subcommands for different modes of operation.

### Primary Commands

```bash
# Run MCP server (stdio) - for AI agent integration
./trindex mcp

# Run HTTP server only - standalone REST API
./trindex server

# Run diagnostics - check configuration and connectivity
./trindex doctor

# Show version information
./trindex version
```

### REST API CLI Commands

When the HTTP server is running, you can use these CLI commands to interact with it:

```bash
# List memories
./trindex memories list
./trindex memories list --namespace myproject --limit 50
./trindex memories list --json

# Get a specific memory
./trindex memories get <id>

# Create a memory
./trindex memories create --content "Important information"
./trindex memories create --content "Project details" --namespace work --metadata key=value

# Delete a memory
./trindex memories delete <id>
./trindex memories delete <id> --force

# Search memories
./trindex search "query"
./trindex search "query" --namespace myproject --top-k 20

# Show statistics
./trindex stats
./trindex stats --json

# Export memories
./trindex export --output memories.jsonl
./trindex export --namespace myproject --since 2024-01-01T00:00:00Z

# Import memories
./trindex import memories.jsonl
./trindex import memories.jsonl --skip-existing
```

### Global Flags

```bash
--config PATH      # Config file path
--env-file PATH    # .env file path
--log-level LEVEL  # Log level (debug|info|warn|error)
--json             # Output as JSON
--api-key KEY      # API key for REST commands
--api-url URL      # Trindex HTTP API URL
```

## Web UI

Trindex includes a built-in web interface for browsing and managing memories. The web UI is automatically served when the HTTP server is running.

### Accessing the Web UI

Once the server is running, open your browser to:

```
http://localhost:9636
```

The web interface provides:

- **Dashboard** - Overview of memory statistics and recent activity
- **Memory Browser** - View, search, create, and delete memories
- **Search** - Perform semantic searches with filters
- **Stats** - Detailed analytics on memory usage

### Web UI Features

- Dark mode toggle
- Responsive design for mobile devices
- Real-time memory statistics
- Namespace filtering
- Similarity-based search results

## Advanced Features

### Deduplication

Trindex provides both client-side and server-side deduplication:

**Client-side (threshold-based):**
```bash
# Skip if similar content already exists
# 0.95 = exact match (content hash), 0.85 = semantic similarity
./trindex memories create \
  --content "Architecture decision: using PostgreSQL" \
  --namespace project:myapp \
  --metadata skip_duplicate_threshold=0.95
```

**Server-side (content hash):**
- Automatically prevents identical content in the same namespace
- Uses SHA-256 hash of normalized content
- Backward compatible with existing memories

### TTL (Time-To-Live)

Set expiration for temporary memories:

```bash
# Expires after 1 hour (3600 seconds)
./trindex memories create \
  --content "Temporary debugging notes" \
  --namespace session:debug-123 \
  --metadata ttl_seconds=3600
```

**Session namespaces** (`session:*`) automatically get 24-hour TTL unless overridden.

### Context Window Ranking

Build optimized context windows for LLM prompts with intelligent ranking:

```go
// Rank memories by relevance (50%) + recency (30%) + importance (20%)
window, err := memory.BuildContextWindow(ctx, "auth implementation", 
    []string{"project:myapp"}, memory.ContextWindowOptions{
        MaxTokens: 4000,
        TopK: 20,
    })
```

**Ranking factors:**
- **Relevance**: Hybrid search similarity score
- **Recency**: Time decay with 24h half-life (newer = higher)
- **Importance**: Type-based boost (decision > bug > outcome > pattern)

### Context Passport

Transfer context between AI systems:

```go
// Export context for handoff to GitHub issue
passport, _ := memory.CreatePassport(ctx, memory.PassportParams{
    SourceNamespace: "project:trindex",
    TargetSystem:    "github:issue-123",
    Query:           "deduplication implementation",
    MaxMemories:     10,
    TTLHours:        24,
})

// Import in target system
imported, _ := memory.ImportPassport(ctx, passportJSON, memory.ImportOptions{
    TargetNamespace: "github:issue-123",
})
```

## MCP Client/Server Architecture

Trindex uses a **client/server** model for MCP integration:

- **`trindex server`** - The full backend (Postgres + Ollama + REST API + Web UI) running on port 9636
- **`trindex mcp`** - A thin proxy client that forwards MCP calls to the server

This design allows multiple AI agents to share a centralized memory server.

```
┌─────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│   AI Agent      │stdio │  trindex mcp     │ HTTP │  trindex server  │
│  (opencode)     │──────▶│  (proxy client)  │──────▶│  :9636           │
└─────────────────┘      └──────────────────┘      └──────────────────┘
                              TRINDEX_URL              │
                                                       ▼
                                               ┌──────────────┐
                                               │  PostgreSQL  │
                                               │  + Ollama    │
                                               └──────────────┘
```

### Configuration

By default, `trindex mcp` connects to `http://localhost:9636`. Set `TRINDEX_URL` to point to a different server:

```bash
# Default (connects to localhost:9636)
./trindex mcp

# Connect to remote server
export TRINDEX_URL=https://brain.example.com
./trindex mcp

# With authentication
export TRINDEX_URL=https://brain.example.com
export TRINDEX_API_KEY=your-secret-key
./trindex mcp
```

### Namespace Defaults

Trindex natively supports multi-tenant isolation via Namespaces. By default, the Trindex MCP tool schemas will explicitly instruct connected AI Agents (OpenCode, Claude Code, Cursor, etc.) to store and recall memories using the `default` namespace unless they are explicitly told otherwise in your system conventions or prompts.

### Hierarchical Namespace Convention

Namespaces follow a hierarchical convention for clear scoping:

```
global > project:{name} > agent:{name} > session:{id}
```

| Namespace | Purpose | Auto-searched |
|-----------|---------|---------------|
| `global` | Cross-agent facts (preferences, identity) | **Always** |
| `project:{name}` | Project-specific knowledge | No |
| `agent:{name}` | Agent-specific optimizations | No |
| `session:{id}` | Ephemeral context (auto-expires 24h) | No |

**Examples:**
```bash
# Global user preference (available to all agents)
./trindex memories create --content "User prefers dark mode" --namespace global

# Project-specific architecture decision
./trindex memories create --content "Using pgvector with HNSW" --namespace project:trindex

# Session debugging info (auto-expires in 24h)
./trindex memories create --content "Error when dimensions mismatch" --namespace session:debug-123
```

### opencode

Add to `~/.config/opencode/opencode.json`:

```json
{
  "mcp": {
    "trindex": {
      "type": "local",
      "command": ["/path/to/trindex", "mcp"],
      "enabled": true
    }
  }
}
```

**Prerequisites:**

- Start the Trindex server first: `docker compose up -d` or `./trindex server`
- Ensure `TRINDEX_URL` points to your server (default: <http://localhost:9636>)
- Set `TRINDEX_API_KEY` if your server requires authentication

### Claude Code (CLI)

```bash
claude mcp add trindex --command "/path/to/trindex mcp"
```

**Prerequisites:**
- Start the Trindex server first: `docker compose up -d` or `./trindex server`
- Ensure `TRINDEX_URL` points to your server (default: http://localhost:9636)
- Set `TRINDEX_API_KEY` if your server requires authentication

### Claude Marketplace Plugin

Trindex is available as a Claude Marketplace plugin with automatic installation and updates. See [Marketplace Plugin Guide](docs/marketplace-plugin-guide.md) for:
- Installation method for Claude desktop app
- Configuration and environment variables
- How to adapt this pattern for other tools

## Development

```bash
# Install git hooks
./scripts/install-hooks.sh

# Run tests
go test ./...

# Run linter
golangci-lint run

# Build Docker image
docker build -t trindex .
```

## License

Business Source License 1.1 - See [LICENSE](LICENSE) for details.

## Architecture

- [AGENT.md](AGENT.md) - Detailed architecture documentation and implementation roadmap
- [Marketplace Plugin Guide](docs/marketplace-plugin-guide.md) - Plugin distribution method and CI/CD integration
