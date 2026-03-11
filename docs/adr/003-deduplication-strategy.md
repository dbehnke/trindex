# ADR-003: Two-Tier Deduplication Strategy

## Status
Accepted

## Context
Agents were storing duplicate memories (5+ copies of "User's name is Dave"), causing storage bloat and retrieval noise. We needed deduplication at both client and server levels.

## Decision
Implement two-tier deduplication:

### Tier 1: Client-Side Semantic Check
- Before `remember`, query with high similarity threshold (0.95 exact, 0.85 semantic)
- If match found, skip storage and return existing memory
- Parameter: `skip_duplicate_threshold` (0.0-1.0)

### Tier 2: Server-Side Hash-Based
- SHA-256 content hash stored in `content_hash` column
- Unique constraint: `(namespace, content_hash)`
- Returns existing ID on conflict instead of error

## Consequences

### Positive
- Client-side prevents unnecessary round-trips
- Server-side guarantees no exact duplicates
- Configurable strictness per use case
- Backward compatible (hash computed for existing memories)

### Negative
- Extra query for client-side check (latency)
- Semantic duplicates ("Dave" vs "User is Dave") not caught by hash
- Race condition window between check and write (mitigated by server-side)

## Implementation Details
```go
// Client-side
existing := Recall(content, namespace, threshold, 1)
if existing[0].Score >= threshold {
    return existing[0] // Skip duplicate
}

// Server-side (hash + unique constraint)
content_hash = SHA256(trim(content))
INSERT ... ON CONFLICT (namespace, content_hash) RETURN id
```

## Alternatives Considered
- **Vector index deduplication**: Catches semantic duplicates but adds write latency
- **Merge strategy**: Combine content on duplicate (complex merge heuristics)
- **Periodic cleanup**: Batch deduplication job (doesn't prevent bloat)
