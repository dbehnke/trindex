# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Memory System Enhancements
- **Content Hash Deduplication** - SHA-256 based server-side deduplication with unique constraint per namespace
- **TTL Support** - Time-to-live for temporary memories with automatic expiration
- **Client-side Deduplication** - `skip_duplicate_threshold` parameter in `remember` tool (0.95 exact, 0.85 semantic)
- **Session Namespace Auto-TTL** - `session:*` namespaces automatically expire after 24 hours

#### Advanced Retrieval
- **Context Window Ranking** - Weighted scoring algorithm for LLM context optimization:
  - Relevance: 50%
  - Recency: 30% (24h half-life)
  - Type boost: 20% (decision > bug > outcome > pattern)
- **Context Passport Pattern** - Cross-system context transfer for Linear/GitHub/agent handoff

#### Observability
- **Structured Logging** - JSON-formatted logs with configurable levels (debug/info/warn/error)
- **Prometheus Metrics** - `/metrics` endpoint with HTTP, database, and MCP operation metrics
- **Request Tracing** - Automatic request ID generation and propagation

#### Documentation
- **OpenAPI Specification** - Complete REST API documentation (`docs/openapi.yaml`)
- **Architecture Decision Records** - 6 ADRs documenting key architectural decisions
- **Contributing Guide** - Comprehensive guide for contributors
- **API Reference** - Detailed API documentation (`docs/api.md`)

### Changed

- **Database Schema v2** - Added `content_hash`, `ttl_seconds`, `expires_at` columns
- **Recall Behavior** - Automatically filters expired memories from search results
- **Namespace Convention** - Documented hierarchical convention (`global > project > agent > session`)

### Deprecated

- None

### Removed

- None

### Fixed

- None

### Security

- None

## [1.0.8] - 2026-03-07

### Added
- MCP tool descriptions improved with vector/fts weight parameters

## [1.0.7] - 2026-03-06

### Added
- Release workflow automation

## [1.0.6] - 2026-03-05

### Added
- Claude plugin marketplace integration
- GoReleaser CI/CD pipeline

## [1.0.0] - 2026-03-01

### Added
- Initial release
- Core MCP tools (remember, recall, forget, list, stats)
- Hybrid search with RRF
- Web UI with Vue.js
- REST API
- Namespace support

---

## Release Notes Template

When creating a new release, use this template:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features

### Changed
- Changes in existing functionality

### Deprecated
- Soon-to-be removed features

### Removed
- Now removed features

### Fixed
- Bug fixes

### Security
- Security improvements
```
