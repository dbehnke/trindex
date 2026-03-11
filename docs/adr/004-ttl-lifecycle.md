# ADR-004: Time-To-Live (TTL) for Memory Lifecycle

## Status
Accepted

## Context
Memories were persisting forever, causing:
1. Storage growth without bound
2. Old session data cluttering retrieval
3. Temporary files and debug info never cleaning up

## Decision
Add TTL (Time-To-Live) support with automatic expiration:

### Schema
- `ttl_seconds`: Integer, TTL in seconds (0 = no expiry)
- `expires_at`: Timestamp, calculated expiration time

### Default TTLs
| Namespace Pattern | TTL | Rationale |
|------------------|-----|-----------|
| `session:*` | 24 hours | Ephemeral debugging context |
| `global` | Never | User preferences persist |
| `project:*` | Never | Project knowledge persists |
| `default` | 30 days | Fallback cleanup |

### Cleanup
- Expired memories filtered from recall results automatically
- `DeleteExpired()` method for batch cleanup
- Background job can be scheduled externally (cron, etc.)

## Consequences

### Positive
- Automatic cleanup of ephemeral data
- Explicit intent at write time (temporary vs permanent)
- Prevents session bloat
- Flexible per-memory TTL override

### Negative
- Accidental data loss if wrong TTL set
- No built-in resurrection of expired memories
- Background job requires external scheduling

## Implementation
```sql
ALTER TABLE memories ADD COLUMN ttl_seconds INTEGER DEFAULT 0;
ALTER TABLE memories ADD COLUMN expires_at TIMESTAMPTZ;

-- Query automatically excludes expired
SELECT * FROM memories 
WHERE (expires_at IS NULL OR expires_at > NOW())
```

## Alternatives Considered
- **Archival tier**: Move old to S3 (complex infrastructure)
- **Relevance-based decay**: Gradual fading based on access (unpredictable)
- **Manual deletion**: Rely on users to clean up (won't happen)
