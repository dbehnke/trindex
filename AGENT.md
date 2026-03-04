# Trindex — AGENT.md
> Persistent semantic memory for AI agents. One brain, every agent.

## Project Overview

**Trindex** is a standalone Go binary that provides persistent, semantic memory for AI agents via the Model Context Protocol (MCP). It is a companion project to **Trinity** (orchestration layer) and shares its naming lineage.

Agents store and recall memories using natural language. Memories are embedded as vectors, stored in Postgres with pgvector, and retrieved via cosine similarity search combined with full-text search (hybrid). Any MCP-compatible agent — opencode, Claude Code, Cursor, custom orchestrators — can plug in via stdio.

### Prior Art
Trindex is architecturally inspired by Nate B. Jones's **OpenBrain** guide (Postgres + pgvector + MCP) and shares philosophical goals with the **Engram** project (agent-agnostic Go binary, MCP stdio). Trindex differentiates through: pgvector semantic search, hybrid retrieval (vector + full-text), namespace scoping with global fallback, multi-namespace recall, JSONB metadata, and an embedded Vue/Tailwind v4 monitoring UI (Phase 2).

---

## Locked Decisions

| Concern | Decision |
|---|---|
| Language | Go |
| MCP SDK | `github.com/modelcontextprotocol/go-sdk` (official) |
| Transport (Phase 1) | stdio only |
| Transport (Phase 2) | HTTP/SSE + stdio |
| Database | Postgres with pgvector extension |
| Vector index | HNSW, cosine distance, tunable via env |
| Search | Hybrid: pgvector cosine + Postgres tsvector, fused with RRF |
| Embeddings | OpenAI-compatible endpoint (agnostic — Ollama, LM Studio, OpenAI, etc.) |
| Namespacing | Namespace string per memory, `global` always included in recall |
| Multi-namespace recall | Supported — pass array of namespaces to recall tool |
| Schema | pgvector + JSONB metadata + tsvector generated column |
| Metadata | Agent-provided only (no LLM extraction in Phase 1) |
| Write pipeline | Parallel: embedding generation + metadata storage via goroutines |
| Deployment | Single Go binary + Docker Compose (Postgres sidecar) |
| Postgres image | `pgvector/pgvector:pg17` |
| Monitoring UI | Phase 2 — embedded Vue + Tailwind v4, compiled and served from Go binary |
| Web transport | HTTP/SSE and monitoring UI land in the same phase |

---

## Environment Configuration

```env
# Postgres
DATABASE_URL=postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable

# Embeddings — OpenAI-compatible endpoint
EMBED_BASE_URL=http://localhost:11434/v1
EMBED_MODEL=nomic-embed-text
EMBED_API_KEY=ollama
EMBED_DIMENSIONS=1536

# MCP
TRANSPORT=stdio

# HNSW index tuning
HNSW_M=16
HNSW_EF_CONSTRUCTION=64
HNSW_EF_SEARCH=40

# Recall defaults
DEFAULT_NAMESPACE=default
DEFAULT_TOP_K=10
DEFAULT_SIMILARITY_THRESHOLD=0.7

# Phase 2 (HTTP/SSE + Web UI)
# HTTP_PORT=8080
# HTTP_HOST=0.0.0.0
```

---

## Database Schema

```sql
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Memories table
CREATE TABLE IF NOT EXISTS memories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace   TEXT NOT NULL DEFAULT 'default',
    content     TEXT NOT NULL,
    embedding   VECTOR(1536),
    metadata    JSONB DEFAULT '{}',
    search_vec  TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', content)) STORED,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW vector index (cosine distance)
CREATE INDEX IF NOT EXISTS memories_embedding_hnsw_idx
    ON memories
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Full-text search index
CREATE INDEX IF NOT EXISTS memories_search_vec_idx
    ON memories USING gin(search_vec);

-- JSONB metadata index
CREATE INDEX IF NOT EXISTS memories_metadata_idx
    ON memories USING gin(metadata);

-- Namespace index
CREATE INDEX IF NOT EXISTS memories_namespace_idx
    ON memories (namespace);

-- Timestamp index
CREATE INDEX IF NOT EXISTS memories_created_at_idx
    ON memories (created_at DESC);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER memories_updated_at
    BEFORE UPDATE ON memories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

---

## Project Structure

```
trindex/
├── cmd/
│   └── trindex/
│       └── main.go                  # Entry point, transport selection
├── internal/
│   ├── config/
│   │   └── config.go                # Env-based config with defaults
│   ├── db/
│   │   ├── db.go                    # Postgres connection pool
│   │   └── migrate.go               # Schema migrations on startup
│   ├── embed/
│   │   └── client.go                # OpenAI-compatible embeddings client
│   ├── memory/
│   │   ├── store.go                 # Remember, forget, list
│   │   ├── recall.go                # Hybrid search (vector + FTS + RRF)
│   │   └── stats.go                 # Stats queries
│   ├── mcp/
│   │   ├── server.go                # MCP server setup, tool registration
│   │   └── tools.go                 # Tool handler implementations
│   ├── testutil/                    # Test utilities for integration tests
│   │   ├── db.go                    # Testcontainers Postgres setup
│   │   └── mock_ollama.go           # Mock embedding server
│   └── web/                         # Phase 2: HTTP server + embedded web UI
│       ├── server.go                # HTTP server with REST API
│       └── dist/                    # Compiled Vue assets, embedded via go:embed
├── web/                             # Phase 2: Vue + Tailwind v4 source (builds to internal/web/dist)
│   ├── src/                         # Vue source files
│   └── dist/                        # Build output (copied to internal/web/dist)
├── docker-compose.yml
├── Dockerfile
├── .env.example
└── AGENT.md
```

---

## MCP Tools

### `remember`
Store a memory with optional namespace and metadata.

**Input:**
```json
{
  "content": "string (required) — the memory text to store",
  "namespace": "string (optional, default: 'default') — scope for this memory",
  "metadata": "object (optional) — arbitrary key/value tags: { agent, project, tags[], source, type }"
}
```

**Behavior:**
- Generate embedding and store to Postgres in parallel via goroutines
- Embedding call and DB insert happen concurrently where possible
- Returns structured confirmation: id, namespace, metadata extracted, timestamp

**Response:**
```json
{
  "id": "uuid",
  "namespace": "default",
  "metadata": { "agent": "opencode", "tags": ["architecture"] },
  "created_at": "2026-03-03T12:00:00Z"
}
```

---

### `recall`
Retrieve memories by semantic similarity using hybrid search.

**Input:**
```json
{
  "query": "string (required) — natural language search query",
  "namespaces": "[]string (optional) — namespaces to search; 'global' always included",
  "top_k": "int (optional, default: 10) — number of results to return",
  "threshold": "float (optional, default: 0.7) — minimum similarity score 0.0-1.0",
  "filter": {
    "since": "RFC3339 timestamp (optional)",
    "until": "RFC3339 timestamp (optional)",
    "tags": "[]string (optional) — match any tag in metadata.tags",
    "source": "string (optional) — match metadata.source"
  }
}
```

**Behavior:**
- Embed the query using the configured embedding endpoint
- Run vector search (cosine similarity via HNSW) in parallel with FTS search (tsvector)
- Fuse results using Reciprocal Rank Fusion (RRF)
- Always include `global` namespace in addition to requested namespaces
- Apply metadata filters via JSONB queries after retrieval
- Return results ranked by fused score

**Response:**
```json
{
  "results": [
    {
      "id": "uuid",
      "content": "...",
      "namespace": "opencode",
      "score": 0.92,
      "metadata": {},
      "created_at": "..."
    }
  ],
  "total": 3,
  "namespaces_searched": ["opencode", "global"]
}
```

---

### `forget`
Delete one or more memories.

**Input:**
```json
{
  "id": "string (optional) — delete single memory by UUID",
  "namespace": "string (optional) — delete all memories in namespace",
  "filter": {
    "before": "RFC3339 timestamp (optional) — delete memories older than this",
    "tags": "[]string (optional) — delete memories matching these tags"
  }
}
```

**Note:** At least one of `id`, `namespace`, or `filter` must be provided. Namespace + filter can be combined. Requires explicit confirmation scope — never deletes without a clear target.

---

### `list`
Browse memories without a semantic query. Useful for inspection and debugging.

**Input:**
```json
{
  "namespace": "string (optional) — filter by namespace",
  "limit": "int (optional, default: 20)",
  "offset": "int (optional, default: 0)",
  "order": "string (optional, default: 'desc') — 'asc' or 'desc' by created_at"
}
```

---

### `stats`
Return memory statistics. Useful for monitoring and the web UI.

**Input:**
```json
{
  "namespace": "string (optional) — scope stats to namespace; omit for global"
}
```

**Response:**
```json
{
  "total_memories": 1024,
  "by_namespace": {
    "default": 400,
    "opencode": 300,
    "global": 200,
    "stellar-breach": 124
  },
  "recent_24h": 42,
  "oldest_memory": "2025-11-01T...",
  "newest_memory": "2026-03-03T...",
  "top_tags": ["architecture", "decision", "bug", "person"],
  "embedding_model": "nomic-embed-text",
  "embed_dimensions": 1536
}
```

---

## Hybrid Search Implementation (RRF)

Both searches run concurrently via goroutines. Results are fused before returning.

```
query
  ├── goroutine A: embed query → pgvector cosine search → ranked list A
  └── goroutine B: to_tsquery → tsvector GIN search   → ranked list B
            ↓
      RRF fusion: score = 1/(k + rank_A) + 1/(k + rank_B)  where k=60
            ↓
      apply metadata filters (JSONB)
            ↓
      return top_k results
```

Memories that appear in both lists score significantly higher. Memories that only appear in one list still surface but rank lower. This handles both semantic queries ("what did I decide about the database layer") and exact queries ("pgvector HNSW").

---

## Namespace Design

- Every memory has a `namespace` string (default: `"default"`)
- The `global` namespace is always searched in recall, regardless of what namespaces are requested
- Agents should write project-specific memories to their own namespace
- Cross-cutting facts (user preferences, personal context) should go in `global`
- Orchestrators can pass multiple namespaces to cast a wide net

**Suggested namespace conventions:**
```
global          — always searched, cross-agent facts
default         — fallback when no namespace specified
opencode        — opencode agent session memories
trinity         — Trinity orchestrator memories
stellar-breach  — project-specific memories
personal        — personal context and preferences
```

---

## Docker Compose

```yaml
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

  trindex:
    build: .
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://trindex:trindex@postgres:5432/trindex?sslmode=disable
      EMBED_BASE_URL: ${EMBED_BASE_URL:-http://host.docker.internal:11434/v1}
      EMBED_MODEL: ${EMBED_MODEL:-nomic-embed-text}
      EMBED_API_KEY: ${EMBED_API_KEY:-ollama}
      EMBED_DIMENSIONS: ${EMBED_DIMENSIONS:-768}
      TRANSPORT: stdio
    stdin_open: true

volumes:
  pgdata:
```

---

## Dockerfile

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o trindex ./cmd/trindex

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/trindex .
ENTRYPOINT ["./trindex"]
```

---

## opencode MCP Config

Add to `opencode.json` or `~/.config/opencode/opencode.json`:

```json
{
  "mcp": {
    "trindex": {
      "type": "local",
      "command": ["trindex", "mcp"],
      "enabled": true
    }
  }
}
```

Or with Docker:

```json
{
  "mcp": {
    "trindex": {
      "type": "local",
      "command": ["docker", "compose", "-f", "/path/to/trindex/docker-compose.yml", "run", "--rm", "-i", "trindex", "mcp"],
      "enabled": true
    }
  }
}
```

---

## Claude Code MCP Config

```bash
claude mcp add trindex --command "trindex mcp"
```

Or if running via Docker Compose:

```bash
claude mcp add trindex -- docker compose -f /path/to/trindex/docker-compose.yml run --rm -i trindex mcp
```

---

## Build & Run

### Prerequisites

- **Go 1.26.0+** (required)
- **Node.js 24 LTS+** (required for web UI builds)
- **Postgres 17** with pgvector extension
- **Task** (go-task) — install via `brew install go-task` or [taskfile.dev](https://taskfile.dev/installation/)

### Quick Start

```bash
# Clone and setup
git clone https://github.com/youruser/trindex
cd trindex
cp .env.example .env
# Edit .env with your embedding endpoint

# Run with Docker Compose (recommended)
task docker:up

# Or build and run locally (requires Postgres running)
task build
./trindex server  # Run HTTP server
./trindex mcp     # Run MCP server

# Run tests
task test

# Check versions
task version:check
```

### Available Tasks

```bash
task --list                    # Show all available tasks

# Development
task build                     # Full build with web UI
task build:fast               # Quick build (uses existing web assets)
task dev                      # Build and run server
task run                      # Run server (builds if needed)

# Testing
task test                     # Run all tests
task test:short               # Run tests (skip integration)
task lint                     # Run golangci-lint
task fmt                      # Format Go code

# Web UI
task web:build                # Build web UI and embed assets
task web:dev                  # Run web UI dev server (hot reload)

# Dependencies
task deps                     # Install all dependencies
task deps:go                  # Download Go modules
task deps:node                # Install Node packages

# Docker
task docker:up                # Start Postgres via Docker Compose
task docker:down              # Stop Docker Compose services
task docker:build             # Build Docker image

# Maintenance
task clean                    # Remove build artifacts
task check                    # Run all checks (fmt, lint, test, build)
```

---

## Implementation Phases

### Phase 1 — Core ✅ COMPLETED

#### 1.1 Foundation — Project Scaffold
- [x] **1.1.1** Create Go module (`go mod init`) with project structure
  - `cmd/trindex/main.go` entry point
  - `internal/` packages: config, db, embed, memory, mcp
  - `.gitignore` for Go projects
- [x] **1.1.2** Create `Dockerfile` with multi-stage build
- [x] **1.1.3** Create `docker-compose.yml` with Postgres + pgvector service
- [x] **1.1.4** Create `.env.example` with all documented environment variables
- [x] **1.1.5** Basic `README.md` with quickstart (can be expanded later)

#### 1.2 Infrastructure — Config & Database
- [x] **1.2.1** Implement `internal/config/config.go` — env-based config with defaults
  - All env vars from Environment Configuration section
  - Validation (required fields, numeric ranges)
  - Sensible defaults for all optional values
- [x] **1.2.2** Implement `internal/db/db.go` — Postgres connection pool
  - `pgx/v5` connection pool setup
  - Connection health check on startup
  - Graceful shutdown
- [x] **1.2.3** Implement `internal/db/migrate.go` — automatic schema migrations
  - Run migrations on startup
  - Create extensions (vector, pg_trgm)
  - Create `memories` table with all columns
  - Create all indexes (HNSW, GIN, etc.)
  - Create `update_updated_at` trigger
- [x] **1.2.4** Add database tests — connection and migration verification

#### 1.3 Core Service — Embeddings Client
- [x] **1.3.1** Implement `internal/embed/client.go` — OpenAI-compatible HTTP client
  - `Embed(text string) ([]float32, error)` method
  - Configurable base URL, model, API key
  - Request/response structs for OpenAI API
  - Error handling with structured errors
- [x] **1.3.2** Add embed client tests with mock server
- [x] **1.3.3** Validate embedding dimensions on startup
  - Query endpoint with test text
  - Compare returned dimensions to `EMBED_DIMENSIONS`
  - Fail fast with clear error if mismatch

#### 1.4 Memory Layer — Store & Recall
- [x] **1.4.1** Define memory models in `internal/memory/models.go`
  - `Memory` struct with all fields
  - `RecallResult` struct with score
  - `Filter` struct for metadata filtering
- [x] **1.4.2** Implement `internal/memory/store.go` — basic CRUD operations
  - `Create(ctx, content, namespace, metadata) (*Memory, error)`
  - `DeleteByID(ctx, id) error`
  - `DeleteByNamespace(ctx, namespace, filter) (int, error)`
  - `List(ctx, namespace, limit, offset, order) ([]Memory, error)`
- [x] **1.4.3** Implement `internal/memory/recall.go` — hybrid search
  - `Recall(ctx, query, namespaces, topK, threshold, filter) ([]RecallResult, error)`
  - Parallel vector search goroutine (cosine similarity via pgvector)
  - Parallel FTS search goroutine (tsvector)
  - RRF fusion with k=60
  - Metadata filtering (JSONB queries)
  - Always include `global` namespace
- [x] **1.4.4** Implement `internal/memory/stats.go` — statistics queries
  - `Stats(ctx, namespace) (*Stats, error)`
  - Total count, by namespace, recent 24h, top tags
- [x] **1.4.5** Add memory layer tests

#### 1.5 MCP Layer — Server & Tools
- [x] **1.5.1** Implement `internal/mcp/server.go` — MCP server setup
  - Initialize official MCP Go SDK server
  - stdio transport only
  - Register all 5 tools
  - Graceful shutdown handling
- [x] **1.5.2** Implement `internal/mcp/tools.go` — tool handlers
  - `remember` tool handler
  - `recall` tool handler
  - `forget` tool handler
  - `list` tool handler
  - `stats` tool handler
- [x] **1.5.3** Wire up `cmd/trindex/main.go`
  - Load config
  - Initialize DB connection
  - Run migrations
  - Validate embedding dimensions
  - Start MCP server
  - Handle signals for graceful shutdown
- [x] **1.5.4** Add end-to-end MCP tests (stdio transport)

#### 1.6 Polish — Documentation & Tooling
- [x] **1.6.1** Expand `README.md` with full setup guide
  - Installation (binary, Docker, source)
  - Configuration reference
  - Embedding endpoint setup (Ollama, etc.)
  - MCP config examples for opencode
  - MCP config examples for Claude Code
  - Troubleshooting section
- [x] **1.6.2** Create `Taskfile.yml` with common tasks (using go-task)
  - `task build`, `task test`, `task run`
  - `task docker:build`, `task docker:up`
  - `task lint`, `task fmt`
  - `task web:build`, `task web:dev`
- [x] **1.6.3** Add GitHub Actions CI workflow
  - Run tests on PR/push with Postgres service
  - Build Docker image
  - Lint with golangci-lint
  - Build and verify web UI
- [x] **1.6.4** Final integration test — full workflow
  - Testcontainers-based integration testing with pgvector
  - Mock Ollama server for deterministic embeddings
  - Works on macOS (Colima) and Linux/GitHub Actions
  - See `plans/integration_testing.md` for full details
  - Test utilities in `internal/testutil/`

### Phase 2 — HTTP + Web UI ✅ COMPLETED

#### 2.1 HTTP Transport
- [x] **2.1.1** Implement transport abstraction layer
  - Web server runs alongside MCP stdio transport
  - Clean separation of concerns between MCP and HTTP APIs
- [x] **2.1.2** REST API implementation
  - Full REST API for all memory operations
  - CORS enabled for web UI access

#### 2.2 Web Server Foundation
- [x] **2.2.1** Set up HTTP server (Chi)
  - Configurable port/host via env vars (`HTTP_HOST`, `HTTP_PORT`)
  - Middleware: logging, recovery, CORS
  - Health check endpoint at `/health`
- [x] **2.2.2** Implement API key authentication middleware
  - `TRINDEX_API_KEY` validation
  - Protected API routes

#### 2.3 REST API Endpoints
- [x] **2.3.1** Implement memories API
  - `GET /api/memories` — list with filters (namespace, limit, offset, order)
  - `GET /api/memories/:id` — get single memory
  - `POST /api/memories` — create memory
  - `DELETE /api/memories/:id` — delete memory
- [x] **2.3.2** Implement search API
  - `POST /api/search` — hybrid search endpoint with RRF fusion
- [x] **2.3.3** Implement stats API
  - `GET /api/stats` — dashboard statistics (counts, namespaces, tags)

#### 2.4 Web UI — Vue + Tailwind v4
- [x] **2.4.1** Set up Vue 3 + Tailwind v4 project in `web/`
  - Vite build setup
  - Tailwind v4 configuration with CSS custom properties
  - Basic app shell (header, nav, main content area)
- [x] **2.4.2** Build Dashboard view
  - Memory count by namespace
  - Quick stats cards (total, 24h, namespaces)
  - Top tags display
- [x] **2.4.3** Build Memory Browser view
  - Paginated memory list
  - Filter by namespace
  - Create new memory modal
  - Delete memory action
- [x] **2.4.4** Build Search view
  - Search input with hybrid results
  - Filter by namespace
  - Result cards with similarity scores
- [x] **2.4.5** Build Stats view
  - Namespace distribution chart
  - Top tags visualization
  - Embedding model info display
- [x] **2.4.6** Integrate compiled assets
  - Build script compiles Vue app to `web/dist`
  - `go:embed` embeds `web/dist` in Go binary
  - Static files served from HTTP server

#### 2.5 Web UI Polish
- [x] **2.5.1** Dark mode support
  - Toggle button in header
  - CSS custom properties for theming
- [x] **2.5.2** Responsive design for mobile
  - Sidebar hidden on mobile
  - Flexible grid layouts
- [x] **2.5.3** Loading states and error handling UI
  - Loading indicators
  - Error messages in modals

### Phase 3 — Polish (in progress)

#### 3.1 Enhanced Features
- [ ] **3.1.1** LLM metadata extraction (optional)
  - Configurable via env var (off by default)
  - Use configured model to extract tags, entities from content
  - Merge with agent-provided metadata
- [x] **3.1.2** Memory import from OpenBrain/Supabase
  - REST API: `POST /api/import` with streaming JSONL support
  - Map OpenBrain schema to Trindex schema
  - Handle embedding dimension mismatches via `regenerate_embeddings` option
- [x] **3.1.3** Memory export for backup/migration
  - REST API: `GET /api/export` with namespace and date filters
  - JSONL format with full metadata
  - Streaming export for large datasets
- [x] **3.1.4** Duplicate detection
  - REST API: `GET /api/duplicates` finds near-identical memories (similarity > 0.95)
  - `POST /api/duplicates/merge` merges duplicate memories
  - Configurable similarity threshold

#### 3.2 Search Improvements
- [x] **3.2.1** Configurable hybrid search weights
  - `HYBRID_VECTOR_WEIGHT` env var (default: 0.7)
  - `HYBRID_FTS_WEIGHT` env var (default: 0.3)
  - Per-query weight override in `recall` tool via `VectorWeight` and `FTSWeight` params
- [ ] **3.2.2** Per-query HNSW tuning
  - `ef_search` parameter in `recall` tool (optional)
  - Override default from env var
- [ ] **3.2.3** HNSW index health monitoring
  - Track index staleness (deleted vectors ratio)
  - Suggest reindex when threshold exceeded
  - CLI command to trigger reindex

#### 3.3 Performance & Reliability
- [x] **3.3.1** Connection pooling tuning
  - `DB_MAX_CONNS`, `DB_MIN_CONNS` env vars
  - `DB_MAX_CONN_LIFETIME_MINUTES`, `DB_MAX_CONN_IDLE_TIME_MINUTES` env vars
- [x] **3.3.2** Embedding client improvements
  - Retry logic with exponential backoff (`EMBED_MAX_RETRIES`, `EMBED_RETRY_DELAY_MS`)
  - Request timeout configuration (`EMBED_REQUEST_TIMEOUT_SEC`)
  - Batch embedding support (already supported)
- [ ] **3.3.3** Observability
  - Structured logging with levels
  - Metrics endpoint (Prometheus format)
  - Request tracing

#### 3.4 Documentation & Community
- [ ] **3.4.1** API documentation (OpenAPI spec)
- [ ] **3.4.2** Architecture decision records (ADRs)
- [ ] **3.4.3** Contributing guide
- [ ] **3.4.4** Changelog and versioning

---

## Key Dependencies

### Runtime Requirements

- **Go 1.26.0+** — Language runtime
- **Node.js 24 LTS+** — For building web UI
- **Task (go-task)** — Build automation (`brew install go-task`)

### Go Dependencies

```go
// go.mod (primary dependencies)
github.com/modelcontextprotocol/go-sdk  // Official MCP SDK
github.com/jackc/pgx/v5                 // Postgres driver with pgvector support
github.com/pgvector/pgvector-go         // pgvector Go types
github.com/google/uuid                  // UUID generation
```

### Node.js Dependencies

```json
// package.json
{
  "engines": {
    "node": ">=24.0.0",
    "npm": ">=10.0.0"
  }
}
```

---

## Error Codes

All MCP tool errors return structured responses:

| Code | Meaning |
|---|---|
| `INVALID_INPUT` | Missing required field or bad type |
| `EMBED_FAILED` | Embedding endpoint unreachable or returned error |
| `DB_UNAVAILABLE` | Postgres connection failed |
| `NOT_FOUND` | Memory ID not found for forget/lookup |
| `NAMESPACE_REQUIRED` | Forget called without sufficient scope |

---

## Notes for Implementation Agent

- Run schema migrations automatically on startup — never require manual SQL
- The `global` namespace inclusion in recall is non-negotiable — always add it to the search scope even if not requested
- Embedding dimensions must match between stored vectors and query vectors — validate on startup and fail fast if mismatch detected
- Use `pgvector/pgvector:pg17` Docker image — plain `postgres:17` does not have the vector extension
- The `tsvector` column is `GENERATED ALWAYS AS` — Postgres maintains it automatically, no application logic needed
- For the parallel write pipeline: embed and metadata prep can run concurrently, but the DB insert waits for the embedding result
- Tool descriptions must be written for both human and LLM readability — the agent decides which tool to call based on the description
- Never delete without explicit scope — `forget` with no filters should return `INVALID_INPUT`, not delete everything
- The official MCP Go SDK API may still be evolving — pin to a specific version in go.mod

---

## Work Unit Quick Reference

**Current Focus**: Phase 3 — Polish (enhancements and refinements)

### Ready to Start (no dependencies)
| Unit | Task | Est. Time |
|------|------|-----------|
| 1.1.1 | Go module + project structure | 15 min |
| 1.1.2 | Dockerfile | 10 min |
| 1.1.3 | docker-compose.yml | 10 min |
| 1.1.4 | .env.example | 10 min |
| 1.1.5 | Basic README.md | 15 min |

### Blocked by 1.1.x
| Unit | Task | Est. Time | Depends On |
|------|------|-----------|------------|
| 1.2.1 | Config package | 30 min | 1.1.1 |
| 1.2.2 | DB connection pool | 30 min | 1.1.1 |
| 1.2.3 | Schema migrations | 45 min | 1.2.2 |
| 1.2.4 | DB tests | 20 min | 1.2.3 |
| 1.3.1 | Embed client | 45 min | 1.1.1 |
| 1.3.2 | Embed client tests | 30 min | 1.3.1 |
| 1.3.3 | Dimension validation | 15 min | 1.3.1, 1.2.1 |

### Blocked by 1.2.x and 1.3.x
| Unit | Task | Est. Time | Depends On |
|------|------|-----------|------------|
| 1.4.1 | Memory models | 15 min | 1.1.1 |
| 1.4.2 | Store (CRUD) | 45 min | 1.4.1, 1.2.3 |
| 1.4.3 | Recall (hybrid search) | 90 min | 1.4.1, 1.2.3, 1.3.1 |
| 1.4.4 | Stats queries | 30 min | 1.2.3 |
| 1.4.5 | Memory layer tests | 30 min | 1.4.2-1.4.4 |
| 1.5.1 | MCP server setup | 30 min | 1.2.1 |
| 1.5.2 | Tool handlers | 60 min | 1.5.1, 1.4.x |
| 1.5.3 | Main.go wiring | 30 min | 1.5.2 |
| 1.5.4 | E2E tests | 30 min | 1.5.3 |

### Blocked by 1.5.x
| Unit | Task | Est. Time | Depends On |
|------|------|-----------|------------|
| 1.6.1 | Full README | 45 min | 1.5.4 |
| 1.6.2 | Taskfile.yml | 15 min | 1.1.1 |
| 1.6.3 | GitHub Actions CI | 20 min | 1.5.4 |
| 1.6.4 | Final integration test | 30 min | 1.6.1-1.6.3 |

**Total Phase 1 Est. Time**: ~11-12 hours of focused work

---

## Definition of Done (per work unit)

Each work unit is complete when:
1. Code compiles without errors (`go build ./...`)
2. Tests pass (`go test ./...`)
3. No lint errors (`golangci-lint run`)
4. Documented (comments, README updates as needed)
5. Committed with descriptive message

---

(End of file)
