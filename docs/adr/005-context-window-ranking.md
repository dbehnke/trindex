# ADR-005: Context Window Ranking for LLM Optimization

## Status
Accepted

## Context
When building prompts for LLMs, we need to fit the most relevant context within token limits. Simple recall by similarity isn't enough—we need to balance:
1. Relevance to current query
2. Recency (newer context is often more important)
3. Importance (decisions > random facts)

## Decision
Implement weighted scoring algorithm for context window ranking:

```
final_score = (relevance * 0.5) + (recency * 0.3) + (type_boost * 0.2)
```

### Components
1. **Relevance (50%)**: Hybrid search similarity score
2. **Recency (30%)**: Time decay with 24h half-life
   - `recency = 1 / (1 + hours_ago / 24)`
3. **Type Boost (20%)**: Importance based on metadata.type
   - `decision`: +0.3
   - `bug`: +0.25
   - `outcome`: +0.2
   - `pattern`: +0.15
   - `preference`: +0.1

### Token Budget Management
- Estimate tokens from content length
- Fit memories until budget exceeded
- Return top N that fit

## Consequences

### Positive
- Optimized context for LLM prompts
- Automatically surfaces important decisions
- Respects token limits
- Better than naive top-k similarity

### Negative
- Complex ranking heuristics (tuning required)
- May drop relevant but older memories
- Non-deterministic (depends on current time)

## Usage
```go
window, err := memory.BuildContextWindow(ctx, "auth query", 
    []string{"project:myapp"},
    memory.ContextWindowOptions{
        MaxTokens: 4000,
        TopK: 20,
    })
```

## Alternatives Considered
- **Pure recency**: Misses important old decisions
- **Pure relevance**: May miss recent context shifts
- **Fixed quotas per type**: Too rigid, doesn't adapt to query
