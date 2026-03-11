# Contributing to Trindex

Thank you for your interest in contributing to Trindex! This guide will help you get started.

## Table of Contents

- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Code Style](#code-style)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)

## Development Setup

### Prerequisites

- **Go 1.26+** - [Installation guide](https://go.dev/doc/install)
- **Docker** - For running PostgreSQL with pgvector
- **Node.js 24+** - For building the web UI
- **Task** (go-task) - Build automation: `brew install go-task`
- **Ollama** - For embeddings (optional, tests use mock)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/dbehnke/trindex.git
cd trindex

# Install dependencies
task deps

# Start PostgreSQL
task docker:up

# Run tests
task test

# Build the binary
task build
```

## Project Structure

```
trindex/
├── cmd/trindex/          # Entry point
├── internal/
│   ├── auth/            # Authentication/authorization
│   ├── cmd/             # CLI command implementations
│   ├── config/          # Configuration management
│   ├── db/              # Database connection and migrations
│   ├── embed/           # Embedding client (OpenAI-compatible)
│   ├── eval/            # Cognitive evaluation suite
│   ├── mcp/             # MCP server and tools
│   ├── memory/          # Core memory layer (store, recall, etc.)
│   ├── observability/   # Logging, metrics, tracing
│   ├── testutil/        # Test utilities
│   └── web/             # HTTP server and web UI
├── web/                  # Vue.js frontend source
├── docs/                 # Documentation
│   ├── adr/             # Architecture Decision Records
│   └── openapi.yaml     # OpenAPI specification
└── plans/               # Design documents and plans
```

## Making Changes

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Your Changes

- Follow existing code patterns
- Add tests for new functionality
- Update documentation as needed

### 3. Run Quality Checks

```bash
# Format code
task fmt

# Run linter
task lint

# Run tests
task test

# Build and verify
task build
```

## Testing

### Unit Tests

```bash
go test ./internal/memory/... -v
```

### Integration Tests

Integration tests use `testcontainers-go` to spin up ephemeral PostgreSQL instances:

```bash
go test ./... -v
```

**Note:** First run may be slow as it downloads the `pgvector:pg17` Docker image.

### Cognitive Evaluation

Run the built-in evaluation suite to verify recall precision:

```bash
task eval
```

## Code Style

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting (enforced by linter)
- Prefer explicit error handling over panics
- Document exported functions and types

### Example

```go
// CreateMemory stores a new memory with the given parameters.
// Returns ErrDuplicate if content already exists in namespace.
func (s *Store) CreateMemory(ctx context.Context, params CreateParams) (*Memory, error) {
    if params.Content == "" {
        return nil, fmt.Errorf("%w: content is required", ErrInvalidInput)
    }
    // ... implementation
}
```

### Web UI

- Vue 3 Composition API
- Tailwind CSS for styling
- Follow existing component patterns

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <description>

[optional body]

[optional footer]
```

### Types

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style (formatting, no logic change)
- `refactor:` - Code refactoring
- `test:` - Test additions/changes
- `chore:` - Build/process changes

### Examples

```
feat: add content hash deduplication

- Add SHA-256 content_hash column
- Unique constraint per namespace
- Return existing ID on duplicate

fix: filter expired memories from recall

docs: update API documentation for TTL support
```

## Pull Request Process

1. **Ensure tests pass**
   ```bash
   task check
   ```

2. **Update documentation**
   - Add ADR for architectural changes
   - Update relevant docs/*.md files
   - Update OpenAPI spec if API changed

3. **Create PR**
   - Use descriptive title
   - Reference any related issues
   - Include summary of changes

4. **Code Review**
   - Address review feedback
   - Keep commits atomic and focused
   - Rebase if requested

5. **Merge**
   - Squash merge to main
   - Ensure CI passes

## Architecture Decision Records

For significant architectural changes, create an ADR in `docs/adr/`:

```markdown
# ADR-XXX: Title

## Status
Proposed / Accepted / Deprecated

## Context
What is the issue we're solving?

## Decision
What did we decide?

## Consequences
What are the trade-offs?

## Alternatives Considered
What else did we consider?
```

## Getting Help

- **GitHub Issues** - Bug reports and feature requests
- **GitHub Discussions** - Questions and ideas
- **Discord** (if applicable) - Real-time chat

## License

By contributing, you agree that your contributions will be licensed under the Business Source License 1.1.

---

**Thank you for contributing to Trindex!**
