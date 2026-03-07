# Trindex — CLAUDE.md

> Persistent semantic memory for AI agents via MCP (Model Context Protocol).

## Session Start Protocol

At the beginning of each session:

1. Call `stats` (no namespace) to see what namespaces exist and how much is stored.
2. Call `recall` 2–3 times with task-relevant queries (e.g. "trindex architecture decisions", "recent bugs fixed", "user preferences for this project") before starting work.
3. This orients you before you write a single line of code.

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

| Namespace | Purpose |
|-----------|---------|
| `global` | Cross-agent user facts: preferences, identity, persistent personal context. Always searched automatically on every recall — you never need to include it explicitly. |
| `default` | Fallback when no project context is clear. |
| `trindex` | Memories specific to the Trindex project itself. |
| `personal` | Personal context about the user that is not project-specific. |

Use `global` for facts that should be available to any agent in any session. Use `trindex` for anything specific to this codebase or its development.

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

## Memory Hygiene

- Before ending a long session, check for duplicate memories with `recall` and use `forget` to prune stale or incorrect ones.
- If you store a memory and later find it was wrong, `forget` it by ID and store a corrected version.
- Use `list` with a namespace to audit what exists before bulk operations.

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
