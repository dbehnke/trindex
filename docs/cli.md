# CLI Reference

Complete reference for the Trindex CLI commands.

## Overview

Trindex provides a command-line interface with explicit subcommands for different modes of operation:

- **mcp** - Run MCP server (stdio) for AI agent integration
- **server** - Run HTTP server only for standalone REST API deployment
- **doctor** - Run diagnostics to check configuration and connectivity
- **memories** - Memory operations (list, get, create, delete)
- **search** - Search memories with semantic similarity
- **stats** - Show database statistics
- **export** - Export memories to JSONL
- **import** - Import memories from JSONL
- **version** - Show version information

## Global Flags

These flags are available for all commands:

| Flag | Description | Default |
|------|-------------|---------|
| `--config PATH` | Config file path | - |
| `--env-file PATH` | .env file path | - |
| `--log-level LEVEL` | Log level (debug\|info\|warn\|error) | info |
| `--json` | Output as JSON | false |
| `--api-key KEY` | API key for REST commands | TRINDEX_API_KEY env |
| `--api-url URL` | Trindex HTTP API URL | http://localhost:8080 |

## Commands

### mcp

Run the MCP server for AI agent integration via stdio transport.

```bash
trindex mcp [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--config PATH` | Config file path | - |
| `--remote URL` | Remote Trindex HTTP API URL (future) | - |
| `--api-key KEY` | API key for remote connection | - |

**Example:**

```bash
# Run MCP server with default config
trindex mcp

# Run with custom config
trindex mcp --config /path/to/config.yaml
```

### server

Run the HTTP server only (no MCP).

```bash
trindex server [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--host HOST` | HTTP host | 0.0.0.0 |
| `--port PORT` | HTTP port | 8080 |
| `--no-ui` | Disable web UI, API only | false |

**Example:**

```bash
# Run server on default port
trindex server

# Run on custom port
trindex server --port 3000

# API only (no web UI)
trindex server --no-ui
```

### doctor

Run diagnostics to check configuration and connectivity.

```bash
trindex doctor
```

**Checks performed:**

1. Configuration loading
2. Database connectivity
3. Embedding endpoint accessibility
4. Dimension validation

**Exit codes:**

- `0` - All checks passed
- `1` - One or more checks failed

**Example:**

```bash
trindex doctor
# Output:
# 🔍 Trindex Doctor
#
# Checking configuration... ✅ PASSED
#    Database URL: postgres://trindex:***@localhost:5432/trindex
#    Embed Model: nomic-embed-text
#    Embed Dimensions: 768
#
# Checking database connection... ✅ PASSED
#    Memories in database: 42
#
# Checking embedding endpoint... ✅ PASSED
#    Endpoint: http://localhost:11434/v1
#    Returned dimensions: 768
#
# 🎉 All checks passed! Trindex is ready to go.
```

### memories

Memory operations with subcommands.

```bash
trindex memories <subcommand> [flags]
```

**Subcommands:**

- `list` - List memories
- `get ID` - Get a memory by ID
- `create` - Create a new memory
- `delete ID` - Delete a memory

#### memories list

List memories with optional filtering.

```bash
trindex memories list [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace NS` | Filter by namespace | - |
| `--limit N` | Limit results | 20 |
| `--offset N` | Pagination offset | 0 |
| `--order asc\|desc` | Sort order | desc |
| `--json` | Output as JSON | false |

**Example:**

```bash
# List all memories
trindex memories list

# List memories in namespace
trindex memories list --namespace myproject

# List with pagination
trindex memories list --limit 50 --offset 100

# Output as JSON
trindex memories list --json
```

#### memories get

Get a specific memory by ID.

```bash
trindex memories get <id> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output as JSON | false |

**Example:**

```bash
trindex memories get 550e8400-e29b-41d4-a716-446655440000
```

#### memories create

Create a new memory.

```bash
trindex memories create [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--content "text"` | Memory content | - |
| `--namespace NS` | Namespace | default |
| `--metadata key=value` | Metadata key-value | - |
| `--file PATH` | Read content from file | - |
| `--json` | Output as JSON | false |

**Example:**

```bash
# Create simple memory
trindex memories create --content "Important information"

# Create with namespace and metadata
trindex memories create \
  --content "Project architecture" \
  --namespace work \
  --metadata project=myapp \
  --metadata type=architecture

# Create from file
trindex memories create --file notes.txt --namespace research
```

#### memories delete

Delete a memory by ID.

```bash
trindex memories delete <id> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Skip confirmation | false |

**Example:**

```bash
# Delete with confirmation
trindex memories delete 550e8400-e29b-41d4-a716-446655440000

# Delete without confirmation
trindex memories delete 550e8400-e29b-41d4-a716-446655440000 --force
```

### search

Search memories using semantic similarity.

```bash
trindex search "query" [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace NS` | Search namespace (repeatable) | - |
| `--top-k N` | Number of results | 10 |
| `--threshold 0.0-1.0` | Similarity threshold | 0.7 |
| `--json` | Output as JSON | false |

**Example:**

```bash
# Simple search
trindex search "machine learning algorithms"

# Search in specific namespace
trindex search "project plans" --namespace work

# Search with options
trindex search "architecture" --top-k 20 --threshold 0.8

# Output as JSON
trindex search "design patterns" --json
```

### stats

Show database statistics.

```bash
trindex stats [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace NS` | Stats for specific namespace | - |
| `--json` | Output as JSON | false |

**Example:**

```bash
# Show all stats
trindex stats

# Stats for namespace
trindex stats --namespace myproject

# Output as JSON
trindex stats --json
```

### export

Export memories to JSONL format.

```bash
trindex export [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--output FILE` | Output file (default: stdout) | - |
| `--namespace NS` | Export specific namespace | - |
| `--since DATE` | Export memories since date (RFC3339) | - |
| `--until DATE` | Export memories until date (RFC3339) | - |

**Example:**

```bash
# Export to stdout
trindex export

# Export to file
trindex export --output memories.jsonl

# Export specific namespace
trindex export --namespace myproject --output myproject.jsonl

# Export date range
trindex export --since 2024-01-01T00:00:00Z --until 2024-12-31T23:59:59Z
```

### import

Import memories from JSONL format.

```bash
trindex import <file> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--skip-existing` | Skip duplicates | false |
| `--namespace NS` | Import to specific namespace | - |

**Example:**

```bash
# Import from file
trindex import memories.jsonl

# Skip existing memories
trindex import memories.jsonl --skip-existing

# Import to specific namespace
trindex import memories.jsonl --namespace work
```

### version

Show version information.

```bash
trindex version
```

## Environment Variables

Trindex can be configured via environment variables. These can also be set in a `.env` file.

### Database

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable |

### Embedding

| Variable | Description | Default |
|----------|-------------|---------|
| `EMBED_BASE_URL` | Embedding API base URL | http://localhost:11434/v1 |
| `EMBED_MODEL` | Embedding model name | nomic-embed-text |
| `EMBED_API_KEY` | Embedding API key | ollama |
| `EMBED_DIMENSIONS` | Embedding dimensions | 768 |

### Server

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_HOST` | HTTP server host | 0.0.0.0 |
| `HTTP_PORT` | HTTP server port | 8080 |
| `TRINDEX_API_KEY` | API key for REST endpoints | - |

### Search

| Variable | Description | Default |
|----------|-------------|---------|
| `DEFAULT_TOP_K` | Default number of search results | 10 |
| `DEFAULT_SIMILARITY_THRESHOLD` | Default similarity threshold | 0.7 |
| `HYBRID_VECTOR_WEIGHT` | Vector search weight | 0.7 |
| `HYBRID_FTS_WEIGHT` | Full-text search weight | 0.3 |

### HNSW Index

| Variable | Description | Default |
|----------|-------------|---------|
| `HNSW_M` | HNSW M parameter | 16 |
| `HNSW_EF_CONSTRUCTION` | HNSW ef_construction | 64 |
| `HNSW_EF_SEARCH` | HNSW ef_search | 40 |

## Configuration File

Trindex supports YAML configuration files. Use `--config` to specify the path.

**Example config.yaml:**

```yaml
database:
  url: postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable

embedding:
  base_url: http://localhost:11434/v1
  model: nomic-embed-text
  api_key: ollama
  dimensions: 768

server:
  host: 0.0.0.0
  port: 8080

recall:
  default_namespace: default
  default_top_k: 10
  default_threshold: 0.7
```
