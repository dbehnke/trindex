# Trindex

Persistent semantic memory for AI agents via MCP (Model Context Protocol).

## Overview

Trindex is a standalone Go binary that provides persistent, semantic memory for AI agents. It stores memories as vectors in Postgres with pgvector, enabling semantic search and hybrid retrieval (vector + full-text search).

## Quick Start

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
- Ensure `TRINDEX_URL` points to your server (default: http://localhost:9636)
- Set `TRINDEX_API_KEY` if your server requires authentication

### Claude Code

```bash
claude mcp add trindex --command "/path/to/trindex mcp"
```

**Note:** The server must be running before the MCP client starts.

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

See [AGENT.md](AGENT.md) for detailed architecture documentation and implementation roadmap.
