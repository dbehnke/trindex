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
| `--config PATH` | Config file path | See [Configuration Files](#configuration-files) |
| `--env-file PATH` | .env file path | - |
| `--log-level LEVEL` | Log level (debug\|info\|warn\|error) | info |
| `--json` | Output as JSON | false |
| `--api-key KEY` | API key for REST commands | TRINDEX_API_KEY env |
| `--api-url URL` | Trindex HTTP API URL | http://localhost:9636 |

## Configuration Files

Trindex supports configuration via YAML files in standard locations (following [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)):

### Config File Locations (in order of precedence)

1. **Explicit path** (highest priority)
   - `--config /path/to/config.yaml` flag
   - `TRINDEX_CONFIG` environment variable

2. **Current directory**
   - `./trindex.yaml`
   - `./.trindex.yaml`

3. **User config directory** (XDG)
   - Linux: `$XDG_CONFIG_HOME/trindex/config.yaml` or `~/.config/trindex/config.yaml`
   - macOS: `~/Library/Application Support/trindex/config.yaml`
   - Windows: `%APPDATA%\trindex\config.yaml`

4. **Legacy locations**
   - `~/.trindex.yaml`
   - `~/.trindex/config.yaml`

5. **System-wide** (Unix only)
   - `/etc/trindex/config.yaml`
   - `/etc/trindex.yaml`

### Example Config File

Create a file at `~/.config/trindex/config.yaml`:

```yaml
# Database connection
database_url: "postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable"

# Embedding configuration
embed_base_url: "http://localhost:11434/v1"
embed_model: "nomic-embed-text"
embed_api_key: "ollama"
embed_dimensions: 768

# HTTP Server settings
http_host: "0.0.0.0"
http_port: "9636"

# Default search settings
default_namespace: "default"
default_top_k: 10
default_similarity_threshold: 0.7
```

### Configuration Precedence

Configuration values are loaded in this order (later overrides earlier):

1. Default values
2. Config file (from standard locations or `--config`)
3. Environment variables
4. Command-line flags

This means environment variables override config file settings, allowing you to use config files for defaults and environment variables for deployment-specific overrides.

## Commands

### mcp

Run the MCP client for AI agent integration via stdio transport.

By default, `trindex mcp` runs as a **thin proxy client** that forwards MCP tool calls to a Trindex server. This enables multiple AI agents to share a centralized memory server.

```bash
trindex mcp [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--config PATH` | Config file path | - |
| `--remote URL` | Trindex server URL | TRINDEX_URL env or http://localhost:9636 |
| `--api-key KEY` | API key for server authentication | TRINDEX_API_KEY env |

**Environment Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `TRINDEX_URL` | URL of the Trindex server | http://localhost:9636 |
| `TRINDEX_API_KEY` | API key for authentication | - |

**Examples:**

```bash
# Run MCP client (connects to http://localhost:9636 by default)
trindex mcp

# Connect to remote server
export TRINDEX_URL=https://brain.example.com
trindex mcp

# Or use flags
trindex mcp --remote https://brain.example.com --api-key secret

# Run in local mode (legacy, requires local Postgres + Ollama)
trindex mcp --remote local
```

**Prerequisites:**
- The Trindex server must be running (`trindex server` or `docker compose up`)
- `TRINDEX_URL` must point to the server (default: http://localhost:9636)
- `TRINDEX_API_KEY` must be set if the server requires authentication

### server

Run the HTTP server only (no MCP).

```bash
trindex server [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--host HOST` | HTTP host | 0.0.0.0 |
| `--port PORT` | HTTP port | 9636 |
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
| `--skip-duplicate-threshold 0.0-1.0` | Skip if similar memory exists | - |
| `--ttl-seconds N` | Time-to-live in seconds (0 = no expiry) | 0 |

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

# Create with deduplication (skip if 95% similar content exists)
trindex memories create \
  --content "Architecture decision: using PostgreSQL" \
  --namespace project:myapp \
  --skip-duplicate-threshold 0.95

# Create with TTL (expires after 1 hour)
trindex memories create \
  --content "Temporary debugging notes" \
  --namespace session:debug-123 \
  --ttl-seconds 3600
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
| `HTTP_PORT` | HTTP server port | 9636 |
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

### TTL and Deduplication

| Variable | Description | Default |
|----------|-------------|---------|
| `DEFAULT_SESSION_TTL` | Default TTL for session namespaces (seconds) | 86400 (24h) |
| `DEFAULT_DEDUP_THRESHOLD` | Default deduplication threshold (0.0-1.0) | 0.95 |

### Database Connection Pool

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_MAX_CONNS` | Maximum database connections | 25 |
| `DB_MIN_CONNS` | Minimum database connections | 5 |
| `DB_MAX_CONN_LIFETIME_MINUTES` | Maximum connection lifetime | 60 |
| `DB_MAX_CONN_IDLE_TIME_MINUTES` | Maximum idle connection time | 30 |

### Embedding Client

| Variable | Description | Default |
|----------|-------------|---------|
| `EMBED_MAX_RETRIES` | Maximum embedding retries | 3 |
| `EMBED_RETRY_DELAY_MS` | Retry delay in milliseconds | 1000 |
| `EMBED_REQUEST_TIMEOUT_SEC` | Request timeout in seconds | 30 |

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
  port: 9636

recall:
  default_namespace: default
  default_top_k: 10
  default_threshold: 0.7
```
