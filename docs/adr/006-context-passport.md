# ADR-006: Context Passport Pattern for Cross-System Handoff

## Status
Accepted

## Context
AI agents need to transfer context between systems (VS Code → Linear, GitHub → Slack) without losing continuity. Each system has its own memory store, creating silos.

## Decision
Implement explicit **Context Passport** pattern:

### Export (Source System)
```go
passport := memory.CreatePassport(ctx, memory.PassportParams{
    SourceNamespace: "project:trindex",
    TargetSystem:    "github:issue-123",
    Query:           "current work context",
    MaxMemories:     10,
    TTLHours:        24,
})
```

### Passport Structure
```json
{
  "version": "1.0",
  "source": "project:trindex",
  "target": "github:issue-123",
  "created_at": "2026-03-07T12:00:00Z",
  "expires_at": "2026-03-08T12:00:00Z",
  "summary": "Working on deduplication feature",
  "key_facts": ["Content hash is SHA-256"],
  "decisions": [{"content": "...", "rationale": "..."}],
  "memory_refs": [{"id": "...", "content": "..."}],
  "metadata": {"agent": "claude-code"}
}
```

### Import (Target System)
```go
imported, err := memory.ImportPassport(ctx, jsonData, memory.ImportOptions{
    TargetNamespace: "github:issue-123",
    PreserveTTL: true,
})
```

## Consequences

### Positive
- No infrastructure changes required
- Systems remain autonomous
- Explicit context sharing (privacy control)
- Portable JSON format
- TTL prevents stale context

### Negative
- Manual agent coordination required
- Context may be stale by import time
- Duplication across systems
- No automatic synchronization

## Future: MCP-First Architecture
The passport pattern is a stepping stone toward full **MCP-First Architecture** (ADR-007), where all systems share Trindex as single source of truth via MCP adapters.

## Alternatives Considered
- **Event-driven federation**: Complex infrastructure (Kafka/NATS)
- **Shared database**: Tight coupling, single point of failure
- **API integrations**: Every pair of systems needs custom integration
