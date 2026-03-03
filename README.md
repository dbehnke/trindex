# Trindex

Persistent semantic memory for AI agents via MCP (Model Context Protocol).

## Overview

Trindex is a standalone Go binary that provides persistent, semantic memory for AI agents. It stores memories as vectors in Postgres with pgvector, enabling semantic search and hybrid retrieval (vector + full-text search).

## Quick Start

### Prerequisites

- Go 1.23+ (for building from source)
- Docker and Docker Compose (recommended)
- Postgres 17 with pgvector extension
- An OpenAI-compatible embedding endpoint (Ollama, LM Studio, OpenAI, etc.)

### Installation

#### Option 1: Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/dbehnke/trindex.git
cd trindex

# Copy environment file
cp .env.example .env

# Edit .env with your embedding endpoint configuration
# For Ollama (default): EMBED_BASE_URL=http://host.docker.internal:11434/v1

# Start Postgres and trindex
docker compose up -d
```

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/dbehnke/trindex.git
cd trindex

# Build the binary
go build -o trindex ./cmd/trindex

# Run with environment variables
DATABASE_URL=postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable \
  EMBED_BASE_URL=http://localhost:11434/v1 \
  EMBED_MODEL=nomic-embed-text \
  ./trindex
```

## MCP Configuration

### opencode

Add to `~/.config/opencode/opencode.json`:

```json
{
  "mcp": {
    "trindex": {
      "type": "local",
      "command": ["/path/to/trindex"],
      "enabled": true
    }
  }
}
```

### Claude Code

```bash
claude mcp add trindex --command /path/to/trindex
```

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
