# Trindex — CLAUDE.md

> Persistent semantic memory for AI agents via MCP (Model Context Protocol).

## Session Start Protocol (REQUIRED)

At the beginning of **every** session, you MUST call `recall` to orient yourself. This is non-optional.

### Standard Session Initialization Pattern

```go
func InitializeSession(agentName, projectName string) SessionContext {
    // 1. Get global user context (always searched automatically)
    globalFacts := Recall("user identity preferences work style", nil, 0.3, 10)
    
    // 2. Get project-specific context
    projectFacts := Recall("project architecture decisions patterns", ["project:"+projectName], 0.3, 10)
    
    // 3. Get recent activity for continuity
    recentActivity := Recall("recent work current tasks", ["project:"+projectName], 0.1, 5, since=24h)
    
    return SessionContext{
        GlobalFacts:    globalFacts,
        ProjectFacts:   projectFacts,
        RecentActivity: recentActivity,
    }
}
```

### Required Queries (Minimum)

Call `recall` at least 2–3 times with these query types:

1. **User Context**: `recall("user preferences identity work style", [], 0.3, 10)`
   - Threshold: 0.3 (moderate precision for user facts)
   - Returns: How user likes to work, their name, preferences

2. **Project Context**: `recall("project architecture decisions", ["project:trindex"], 0.3, 10)`
   - Threshold: 0.3 (moderate precision for architecture)
   - Returns: Key decisions, patterns, codebase structure

3. **Recent Activity**: `recall("recent work bugs fixed", ["project:trindex"], 0.1, 5)` with since=24h
   - Threshold: 0.1 (wide net for recency-filtered results)
   - Returns: What you were working on recently

### Why This Matters

- Prevents repeating the same questions to the user
- Maintains continuity between sessions
- Surfaces relevant context before you start coding
- Reduces "context loss" errors

### Threshold Guidelines

| Query Type | Threshold | Rationale |
|------------|-----------|-----------|
| User facts | 0.3–0.5 | Precise matches for preferences |
| Architecture | 0.3 | Balance precision/recall |
| Recent work | 0.1 | Wide net filtered by time |
| Exact lookup | 0.7+ | Specific function names, error codes |
| Exploration | 0.0001 | Broad discovery mode |

## When to Remember

Store a memory when you:
- Make a significant architectural or implementation decision
- Learn a user preference or working style
- Identify a recurring pattern or root cause
- Complete a meaningful task or resolve a bug
- Discover a non-obvious fact about this codebase

Do **not** store trivial, ephemeral, or easily re-derivable facts.

Write content as 1–3 concise sentences stating the fact directly. Example:
> "The MCP tools are defined in internal/mcp/tools.go. Each tool has a Description field that AI agents use to decide when and how to call it."

## Namespace Conventions

Namespaces follow a **hierarchical convention** for clear scoping and automatic inheritance:

```
global > project:{name} > agent:{name} > session:{id}
```

| Namespace | Purpose | Auto-searched |
|-----------|---------|---------------|
| `global` | Cross-agent user facts: preferences, identity, persistent personal context. | Always |
| `project:{name}` | Project-specific memories: architecture, decisions, patterns (e.g., `project:trindex`). | No |
| `agent:{name}` | Agent-specific learnings and optimizations (e.g., `agent:claude-code`). | No |
| `session:{id}` | Ephemeral session context (auto-expires after 24h). | No |
| `default` | Fallback when no project context is clear. | No |

### Namespace Selection Rules

1. **Use `global`** for cross-cutting user facts that any agent should know:
   - User preferences ("User prefers dark mode")
   - Identity facts ("User's name is Dave")
   - Persistent personal context

2. **Use `project:{name}`** for project-specific knowledge:
   - Architecture decisions
   - Code patterns discovered
   - Bug root causes
   - Implementation notes

3. **Use `session:{id}`** for temporary context:
   - Current debugging session
   - Temporary file paths
   - Transient errors
   - These auto-expire (TTL = 24h by default)

4. **Avoid `default`** — be explicit about scope.

## Metadata Conventions

| Field | Type | Example values |
|-------|------|----------------|
| `type` | string | `"decision"`, `"preference"`, `"pattern"`, `"bug"`, `"outcome"`, `"fact"` |
| `tags` | []string | `["architecture", "mcp", "search"]` |
| `agent` | string | `"claude-code"`, `"opencode"` |
| `project` | string | `"trindex"` |
| `source` | string | `"session"`, `"user-statement"`, `"code-review"` |

## Recall Strategy

- **Default threshold 0.0001** — wide net, returns loosely related results. Good for orientation.
- **Threshold 0.005–0.02** — higher precision, only close matches. Use when you want specific facts.
- **vector_weight high (e.g. 0.9)** — better for conceptual or paraphrased queries.
- **fts_weight high (e.g. 0.9)** — better for exact terms: function names, error codes, identifiers.
- Always search `["trindex"]` namespace for project-specific memories; `global` is included automatically.
- Pass multiple namespaces (`["trindex", "personal"]`) to cast a wider net across contexts.
- Use `skip_duplicate_threshold` on `remember` to avoid storing duplicates (0.95 for exact, 0.85 for semantic).

## Memory Hygiene

- Before ending a long session, check for duplicate memories with `recall` and use `forget` to prune stale or incorrect ones.
- If you store a memory and later find it was wrong, `forget` it by ID and store a corrected version.
- Use `list` with a namespace to audit what exists before bulk operations.

## Memory Lifecycle (TTL)

Memories can have automatic expiration to prevent storage bloat:

| Namespace Pattern | Default TTL | Rationale |
|------------------|-------------|-----------|
| `session:*` | 24 hours | Ephemeral session context |
| `global` | Never | User preferences persist |
| `project:*` | Never | Project knowledge persists |
| `default` | 30 days | Fallback cleanup |

Use the `ttl_seconds` parameter on `remember` to set explicit expiration for temporary context.

## Key Architectural Facts

| Fact | Detail |
|------|--------|
| Hybrid search | Vector (cosine via pgvector HNSW) + Full-text (tsvector) fused with RRF (k=60) |
| Global namespace | Always searched automatically in `recall`, regardless of requested namespaces |
| Default threshold | `0.0001` (very permissive) |
| Default weights | `vector_weight=0.7`, `fts_weight=0.3` (server config) |
| MCP tool names | `remember`, `recall`, `forget`, `list`, `stats` — never rename these |
| Main tools file | `internal/mcp/tools.go` |
| Memory layer | `internal/memory/recall.go` (hybrid search), `internal/memory/store.go` (CRUD) |
| Config | `internal/config/config.go` — all env-based with defaults |
| Deduplication | Content hash (SHA-256) + semantic similarity threshold |
| TTL | `ttl_seconds` param, `session:*` defaults to 24h |
| Context window | `internal/memory/context_window.go` — relevance/recency/type weighted ranking |
| Passport | `internal/memory/passport.go` — cross-system context transfer |
