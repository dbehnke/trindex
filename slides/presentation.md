# Trindex: Persistent Semantic Memory for AI Agents

---

## What is Trindex?

**Trindex** is a standalone Go binary that provides persistent, semantic memory for AI agents via the Model Context Protocol (MCP).

- Store memories as vector embeddings
- Retrieve via semantic similarity search
- Hybrid search: vector + full-text (RRF fusion)
- Namespace scoping with global fallback
- REST API + Web UI + CLI

---

## The Problem

### AI Agents Have No Memory

- Every conversation starts fresh
- Context windows are limited
- No way to persist knowledge across sessions
- Building custom memory systems is complex

### Existing Solutions

- **LangChain**: Python-only, heavy dependencies
- **OpenBrain**: Great architecture, but requires manual setup
- **Vector DBs**: Powerful but low-level (Pinecone, Weaviate)

**Trindex**: Standalone, language-agnostic, MCP-native

---

## Key Features

### 1. MCP-Native Integration
```json
{
  "mcp": {
    "trindex": {
      "command": ["trindex", "mcp"]
    }
  }
}
```

Works with: Claude Code, opencode, Cursor, any MCP client

### 2. Semantic Search
- Cosine similarity via pgvector
- Full-text search with PostgreSQL tsvector
- Reciprocal Rank Fusion (RRF) for hybrid results

### 3. Namespace Organization
- Isolate memories by project/context
- Global namespace always included
- Multi-namespace recall support

---

## Architecture

```
┌─────────────────┐      ┌─────────────────────┐
│   MCP Client    │stdio │   trindex mcp       │
│ (Claude/opencode│─────▶│   (proxy client)    │
└─────────────────┘      └──────────┬──────────┘
                                     │ HTTP/JSON
                                     ▼
                          ┌─────────────────────┐
                          │   trindex server    │
                          │   (HTTP + Web UI)   │
                          │   :9636             │
                          └──────────┬──────────┘
                                     │
                  ┌──────────────────┴──────────────────┐
                  │  /api/mcp/tools  /api/mcp/call      │
                  │  /api/memories   /api/search         │
                  └──────────────────┬──────────────────┘
                                     │
                          ┌─────────────────────┐
                          │ PostgreSQL +        │
                          │ pgvector            │
                          └─────────────────────┘
```

---

## CLI Redesign: Before & After

### Before (Monolithic)
```bash
./trindex  # Starts everything: MCP + HTTP + DB
# No way to run just HTTP server
# No CLI access to REST API
# No diagnostics
```

### After (Explicit Subcommands)
```bash
./trindex mcp           # MCP proxy client (stdio -> HTTP)
./trindex server        # HTTP server only
./trindex doctor        # Diagnostics
./trindex memories list # CLI access to API
./trindex search "..."  # Search from CLI
./trindex export        # Export memories
```

---

## CLI Commands Demo

### Server Management
```bash
# Run diagnostics
./trindex doctor
# ✅ Config
# ✅ Database connection
# ✅ Embedding endpoint

# Start HTTP server
./trindex server --port 9636

# Start MCP proxy client
./trindex mcp
```

### Memory Operations
```bash
# List memories
./trindex memories list --namespace work --json

# Create memory
./trindex memories create \
  --content "Project architecture" \
  --namespace work \
  --metadata project=myapp

# Search
./trindex search "architecture patterns" \
  --namespace work --top-k 10
```

---

## Technical Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.26+ |
| Database | PostgreSQL 17 + pgvector |
| Vector Index | HNSW (cosine distance) |
| Search | Hybrid: pgvector + tsvector + RRF |
| Embeddings | OpenAI-compatible API |
| Web Framework | Chi router |
| UI | Vue 3 + Tailwind v4 |
| Testing | testcontainers-go |

---

## Database Schema

```sql
CREATE TABLE memories (
    id          UUID PRIMARY KEY,
    namespace   TEXT NOT NULL DEFAULT 'default',
    content     TEXT NOT NULL,
    embedding   VECTOR(768),
    metadata    JSONB DEFAULT '{}',
    search_vec  TSVECTOR GENERATED ALWAYS AS 
                  (to_tsvector('english', content)) STORED,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- HNSW index for fast similarity search
CREATE INDEX memories_embedding_hnsw_idx
    ON memories USING hnsw (embedding vector_cosine_ops);
```

---

## Why Trindex?

### For AI Agent Developers
- Drop-in MCP memory
- No Python dependencies
- Language-agnostic (any MCP client)

### For DevOps
- Single binary deployment
- Docker Compose ready
- PostgreSQL backend (existing infra)

### For End Users
- Web UI for browsing memories
- CLI for scripting
- Import/export for backups

---

## Future Roadmap

### Phase 3: Enterprise Features
- Authentication & RBAC
- Multi-tenant support
- Audit logging
- Memory versioning

### Phase 4: Advanced Search
- Reranking with cross-encoders
- Query expansion
- Automatic namespace detection
- Memory decay/refresh

### Phase 5: Ecosystem
- LangChain integration
- Python client
- Webhook support
- Memory sharing between agents

---

## Getting Started

```bash
# Clone and setup
git clone https://github.com/dbehnke/trindex.git
cd trindex
cp .env.example .env
# Edit .env with your embedding endpoint

# Run with Docker Compose
docker compose up -d

# Run MCP client (defaults to http://localhost:9636)
./trindex mcp

# Or build and run locally
go build -o trindex ./cmd/trindex
./trindex doctor
./trindex server
```

---

## Demo Time

### Live Demonstration

1. Run diagnostics
2. Start server
3. Create memories via CLI
4. Search memories
5. View in Web UI
6. Export and import

---

## Questions?

### Resources
- GitHub: github.com/dbehnke/trindex
- Documentation: docs/cli.md
- MCP Spec: modelcontextprotocol.io

### Contact
- Open an issue on GitHub
- Business Source License 1.1

---

## Thank You!

**Trindex**: One brain, every agent.

Persistent semantic memory for the AI age.
