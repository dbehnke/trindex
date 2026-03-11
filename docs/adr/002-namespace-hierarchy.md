# ADR-002: Namespace Hierarchy Convention

## Status
Accepted

## Context
Multiple AI agents need to share a single memory system while maintaining isolation and discoverability. We needed a convention for organizing memories that allows:
1. Agent-specific context (preferences, project knowledge)
2. Cross-agent sharing (user identity, global facts)
3. Temporary/ephemeral data (debugging sessions)

## Decision
Implement a hierarchical namespace convention:

```
global > project:{name} > agent:{name} > session:{id}
```

### Rules
- **`global`**: Always searched automatically. Cross-agent facts (user preferences, identity)
- **`project:{name}`**: Project-specific knowledge (architecture, decisions)
- **`agent:{name}`**: Agent-specific optimizations (rarely used)
- **`session:{id}`**: Ephemeral context (debugging, temporary files). Auto-expires after 24h
- **`default`**: Fallback when no specific scope is clear

## Consequences

### Positive
- Clear mental model for developers
- Automatic scoping (recall searches up hierarchy)
- Prevents namespace pollution
- Session auto-cleanup via TTL

### Negative
- Requires discipline (agents must use correct namespace)
- Migration pain for existing memories
- Some use cases don't fit hierarchy (cross-project concerns)

## Alternatives Considered
- **Flat namespaces with tags**: More flexible but harder to reason about
- **Automatic classification**: Would require ML/LLM integration, adds latency
- **UUID-based namespaces**: No semantic meaning, hard to query

## Implementation Notes
- `global` namespace automatically included in all recall queries
- Session namespaces (`session:*`) default to 24h TTL
- Metadata conventions defined in CLAUDE.md
