# ADR-001: Hybrid Search with Reciprocal Rank Fusion (RRF)

## Status
Accepted

## Context
Trindex needs to provide both semantic (meaning-based) and lexical (keyword-based) search capabilities. Users may search with natural language queries ("what did we decide about the database") or exact terms ("pgvector HNSW").

## Decision
We will implement **hybrid search** combining:
1. **Vector search** (cosine similarity via pgvector HNSW index) for semantic matching
2. **Full-text search** (PostgreSQL tsvector) for exact keyword matching
3. **Reciprocal Rank Fusion (RRF)** to combine results with k=60

Formula: `score = 1/(k + rank_vector) + 1/(k + rank_fts)`

## Consequences

### Positive
- Handles both semantic and exact queries effectively
- Memories appearing in both result sets rank significantly higher
- No need for users to choose search type
- Configurable weights per query (vector_weight, fts_weight)

### Negative
- Double query execution (can parallelize)
- Requires maintaining both HNSW and GIN indexes
- RRF parameter k=60 is fixed; tuning requires experimentation

## Alternatives Considered
- **Pure vector search**: Misses exact matches, struggles with specific terms
- **Pure FTS**: No semantic understanding, poor for conceptual queries
- **Weighted linear combination**: RRF performs better for heterogeneous ranking

## References
- [Reciprocal Rank Fusion explained](https://plg.uwaterloo.ca/~gvcormac/cormackECIR09.pdf)
