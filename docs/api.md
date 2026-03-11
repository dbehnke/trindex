# Trindex API Reference

Complete API reference for Trindex memory operations.

## Table of Contents

- [MCP Tools](#mcp-tools)
- [REST API](#rest-api)
- [Advanced Features](#advanced-features)
- [Go SDK](#go-sdk)

---

## MCP Tools

Trindex exposes 5 MCP tools for AI agent integration.

### `remember`

Store a memory with optional namespace, metadata, deduplication, and TTL.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `content` | string | ✅ | - | The memory text to store |
| `namespace` | string | ❌ | `"default"` | Scope for this memory |
| `metadata` | object | ❌ | `{}` | Arbitrary key/value tags |
| `skip_duplicate_threshold` | float | ❌ | - | Skip if similar memory exists (0.95=exact, 0.85=semantic) |
| `ttl_seconds` | int | ❌ | `0` | Time-to-live in seconds (0=no expiry) |

**Example Request:**

```json
{
  "content": "Using pgvector with HNSW index for semantic search",
  "namespace": "project:trindex",
  "metadata": {
    "type": "decision",
    "tags": ["architecture", "database"],
    "agent": "claude-code"
  },
  "skip_duplicate_threshold": 0.95,
  "ttl_seconds": 0
}
```

**Example Response (new memory):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "namespace": "project:trindex",
  "content_hash": "a3f5c2...",
  "metadata": {
    "type": "decision",
    "tags": ["architecture", "database"],
    "agent": "claude-code"
  },
  "created_at": "2026-03-07T12:00:00Z",
  "expires_at": null
}
```

**Example Response (duplicate found):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "namespace": "project:trindex",
  "skipped": true,
  "reason": "duplicate_content",
  "similarity": 0.98,
  "created_at": "2026-03-06T10:00:00Z"
}
```

**Notes:**
- Session namespaces (`session:*`) default to 24h TTL unless overridden
- Content hash is computed using SHA-256 of normalized (trimmed) content
- Deduplication checks both exact hash matches and semantic similarity

---

### `recall`

Retrieve memories by semantic similarity using hybrid search.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | ✅ | - | Natural language search query |
| `namespaces` | []string | ❌ | `["default"]` | Namespaces to search (`global` always included) |
| `top_k` | int | ❌ | `10` | Number of results to return |
| `threshold` | float | ❌ | `0.7` | Minimum similarity score (0.0-1.0) |
| `vector_weight` | float | ❌ | `0.7` | Weight for vector search (0.0-1.0) |
| `fts_weight` | float | ❌ | `0.3` | Weight for full-text search (0.0-1.0) |
| `filter` | object | ❌ | `{}` | Metadata filters |

**Filter Object:**

| Field | Type | Description |
|-------|------|-------------|
| `since` | string (RFC3339) | Only memories created after this time |
| `until` | string (RFC3339) | Only memories created before this time |
| `tags` | []string | Match any tag in `metadata.tags` |
| `source` | string | Match `metadata.source` |

**Example Request:**

```json
{
  "query": "database architecture decisions",
  "namespaces": ["project:trindex", "project:myapp"],
  "top_k": 5,
  "threshold": 0.6,
  "vector_weight": 0.8,
  "fts_weight": 0.2,
  "filter": {
    "since": "2026-01-01T00:00:00Z",
    "tags": ["architecture"]
  }
}
```

**Example Response:**

```json
{
  "results": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "content": "Using pgvector with HNSW index for semantic search",
      "namespace": "project:trindex",
      "score": 0.92,
      "metadata": {
        "type": "decision",
        "tags": ["architecture", "database"]
      },
      "created_at": "2026-03-07T12:00:00Z"
    }
  ],
  "total": 1,
  "namespaces_searched": ["project:trindex", "project:myapp", "global"]
}
```

**Notes:**
- Expired memories are automatically filtered out
- The `global` namespace is always included in searches
- Hybrid scores are fused using Reciprocal Rank Fusion (RRF) with k=60

---

### `forget`

Delete one or more memories.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | ❌ | Delete single memory by UUID |
| `namespace` | string | ❌ | Delete all memories in namespace |
| `filter` | object | ❌ | Filter criteria for bulk deletion |

**Filter Object:**

| Field | Type | Description |
|-------|------|-------------|
| `before` | string (RFC3339) | Delete memories older than this |
| `tags` | []string | Delete memories matching these tags |

**Example Requests:**

```json
// Delete by ID
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}

// Delete entire namespace
{
  "namespace": "session:debug-123"
}

// Delete old session memories
{
  "namespace": "session:*",
  "filter": {
    "before": "2026-03-01T00:00:00Z"
  }
}
```

**Example Response:**

```json
{
  "deleted": 5,
  "namespace": "session:debug-123"
}
```

**Notes:**
- At least one of `id`, `namespace`, or `filter` must be provided
- `namespace` and `filter` can be combined
- Use wildcards with caution — `session:*` matches all session namespaces

---

### `list`

Browse memories without a semantic query. Useful for inspection and debugging.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `namespace` | string | ❌ | - | Filter by namespace |
| `limit` | int | ❌ | `20` | Maximum results |
| `offset` | int | ❌ | `0` | Pagination offset |
| `order` | string | ❌ | `"desc"` | Sort order: `"asc"` or `"desc"` by created_at |

**Example Request:**

```json
{
  "namespace": "project:trindex",
  "limit": 10,
  "offset": 0,
  "order": "desc"
}
```

**Example Response:**

```json
{
  "memories": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "content": "Using pgvector with HNSW index",
      "namespace": "project:trindex",
      "metadata": {},
      "created_at": "2026-03-07T12:00:00Z"
    }
  ],
  "total": 42,
  "limit": 10,
  "offset": 0
}
```

---

### `stats`

Return memory statistics. Useful for monitoring and the web UI.

**Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `namespace` | string | ❌ | - | Scope stats to namespace; omit for global |

**Example Response:**

```json
{
  "total_memories": 1024,
  "by_namespace": {
    "default": 400,
    "project:trindex": 300,
    "global": 200,
    "session:debug-123": 124
  },
  "recent_24h": 42,
  "expiring_24h": 5,
  "oldest_memory": "2025-11-01T00:00:00Z",
  "newest_memory": "2026-03-07T12:00:00Z",
  "top_tags": ["architecture", "decision", "bug", "pattern"],
  "embedding_model": "nomic-embed-text",
  "embed_dimensions": 768
}
```

---

## REST API

When running `trindex server`, a REST API is available on port 9636.

### Authentication

All API endpoints (except health check) require an API key:

```bash
curl -H "Authorization: Bearer $TRINDEX_API_KEY" \
  http://localhost:9636/api/memories
```

### Endpoints

#### Memories

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/memories` | List memories |
| GET | `/api/memories/:id` | Get memory by ID |
| POST | `/api/memories` | Create memory |
| DELETE | `/api/memories/:id` | Delete memory |

#### Search & Stats

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/search` | Hybrid search |
| GET | `/api/stats` | Get statistics |

#### Import/Export

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/export` | Export memories (JSONL) |
| POST | `/api/import` | Import memories (JSONL) |

### POST /api/memories

Create a new memory via REST API.

**Request Body:**

```json
{
  "content": "Memory content",
  "namespace": "project:myapp",
  "metadata": {
    "type": "decision",
    "tags": ["architecture"]
  },
  "skip_duplicate_threshold": 0.95,
  "ttl_seconds": 3600
}
```

**Response:** Same as MCP `remember` tool.

### POST /api/search

Search memories using hybrid search.

**Request Body:**

```json
{
  "query": "database architecture",
  "namespaces": ["project:myapp"],
  "top_k": 10,
  "threshold": 0.7,
  "filter": {
    "tags": ["architecture"]
  }
}
```

**Response:** Same as MCP `recall` tool.

### GET /api/export

Export memories to JSONL format.

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `namespace` | Filter by namespace |
| `since` | Export memories since date (RFC3339) |
| `until` | Export memories until date (RFC3339) |

**Example:**

```bash
curl -H "Authorization: Bearer $TRINDEX_API_KEY" \
  "http://localhost:9636/api/export?namespace=project:myapp&since=2026-01-01T00:00:00Z" \
  > myapp-memories.jsonl
```

### POST /api/import

Import memories from JSONL format.

**Request Body:** Streaming JSONL with one memory per line:

```jsonl
{"content": "Memory 1", "namespace": "project:myapp", "metadata": {}}
{"content": "Memory 2", "namespace": "project:myapp", "metadata": {}}
```

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `skip_existing` | Skip duplicates (default: false) |
| `namespace` | Import to specific namespace |

---

## Advanced Features

### Context Window Ranking

Build an optimized context window for LLM prompts.

**Go SDK:**

```go
import "github.com/dbehnke/trindex/internal/memory"

window, err := memory.BuildContextWindow(ctx, 
    "query about authentication", 
    []string{"project:myapp"},
    memory.ContextWindowOptions{
        MaxTokens: 4000,
        TopK: 20,
        Threshold: 0.5,
    },
)

for _, item := range window.Items {
    fmt.Printf("[%s] %s (score: %.3f, tokens: %d)\n",
        item.Memory.Namespace,
        item.Memory.Content,
        item.Score,
        item.Tokens,
    )
}
```

**Ranking Algorithm:**

```
final_score = (relevance * 0.5) + (recency * 0.3) + (type_boost * 0.2)

Where:
- relevance = hybrid search similarity score
- recency = 1 / (1 + hours_ago/24)  // 24h half-life
- type_boost = based on metadata.type:
    - decision: +0.3
    - bug: +0.25
    - outcome: +0.2
    - pattern: +0.15
    - preference: +0.1
```

### Context Passport

Transfer context between AI systems.

**Creating a Passport:**

```go
passport, err := memory.CreatePassport(ctx, memory.PassportParams{
    SourceNamespace: "project:trindex",
    TargetSystem:    "github:issue-123",
    Query:           "deduplication implementation",
    MaxMemories:     10,
    TTLHours:        24,
})

// Serialize for transfer
jsonData, _ := json.Marshal(passport)
```

**Passport Structure:**

```json
{
  "version": "1.0",
  "source": "project:trindex",
  "target": "github:issue-123",
  "created_at": "2026-03-07T12:00:00Z",
  "expires_at": "2026-03-08T12:00:00Z",
  "summary": "Working on deduplication feature",
  "key_facts": ["Content hash is SHA-256", "Session TTL is 24h"],
  "decisions": [
    {
      "content": "Use 0.95 threshold for exact dedup",
      "rationale": "Balances precision vs recall"
    }
  ],
  "memory_refs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "namespace": "project:trindex",
      "content": "..."
    }
  ],
  "metadata": {
    "agent": "claude-code",
    "session_id": "debug-2026-03-07"
  }
}
```

**Importing a Passport:**

```go
imported, err := memory.ImportPassport(ctx, jsonData, memory.ImportOptions{
    TargetNamespace: "github:issue-123",
    PreserveTTL: true,
})
```

---

## Go SDK

### Memory Store

```go
import "github.com/dbehnke/trindex/internal/memory"

// Create memory with options
params := memory.CreateParams{
    Content:            "Memory content",
    Namespace:          "project:myapp",
    Metadata:           map[string]interface{}{
        "type": "decision",
        "tags": []string{"architecture"},
    },
    SkipIfDuplicate:    true,
    DuplicateThreshold: 0.95,
    TTLSeconds:         86400,
}

mem, err := store.CreateWithParams(ctx, params)
```

### Recall with Filters

```go
results, err := store.Recall(ctx, "database query", 
    []string{"project:myapp"},
    10,      // top_k
    0.7,     // threshold
    &memory.Filter{
        Since: time.Now().AddDate(0, -1, 0),
        Tags:  []string{"architecture"},
    },
)
```

### Delete Expired Memories

```go
// Delete all expired memories (where expires_at < NOW())
deleted, err := store.DeleteExpired(ctx)
```

---

## Error Codes

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `INVALID_INPUT` | 400 | Missing required field or bad type |
| `EMBED_FAILED` | 503 | Embedding endpoint unreachable |
| `DB_UNAVAILABLE` | 503 | Postgres connection failed |
| `NOT_FOUND` | 404 | Memory ID not found |
| `NAMESPACE_REQUIRED` | 400 | Forget called without scope |
| `DUPLICATE_CONTENT` | 409 | Content already exists |
| `PASSPORT_EXPIRED` | 410 | Context passport has expired |
| `PASSPORT_INVALID` | 400 | Invalid passport format |

---

## Best Practices

### Namespace Organization

```
global/           # User preferences, cross-agent facts
project:trindex/  # Project-specific knowledge
agent:claude/     # Agent-specific optimizations
session:debug-*/  # Ephemeral debugging context (auto-expires)
```

### Threshold Guidelines

| Use Case | Threshold | Notes |
|----------|-----------|-------|
| Initial orientation | 0.0001 | Wide net, filtered by time |
| General recall | 0.3-0.5 | Balance precision/recall |
| Specific lookup | 0.7+ | Exact matches only |
| Deduplication (exact) | 0.95 | Content hash match |
| Deduplication (fuzzy) | 0.85 | Semantic similarity |

### Memory Types

Use `metadata.type` to improve context window ranking:

- `decision` — Architecture or design decisions (+0.3 boost)
- `bug` — Bug fixes and root causes (+0.25 boost)
- `outcome` — Task completions (+0.2 boost)
- `pattern` — Code patterns discovered (+0.15 boost)
- `preference` — User preferences (+0.1 boost)
- `fact` — General knowledge (no boost)

### TTL Recommendations

| Context Type | Recommended TTL | Rationale |
|--------------|-----------------|-----------|
| Session debug | 1-24 hours | Transient debugging info |
| Temporary files | 1-7 days | Build artifacts, temp paths |
| Working notes | 7-30 days | Active development context |
| Decisions | Never | Permanent architecture records |
| Preferences | Never | User preferences persist |
