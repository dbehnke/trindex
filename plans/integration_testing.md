# Integration Testing Plan

> Testcontainers-based integration testing for Trindex with pgvector + mock Ollama.

## Overview

This plan implements comprehensive integration tests using [Testcontainers for Go](https://gopkg.in/testcontainers/testcontainers-go) to spin up real Postgres + pgvector containers for testing. Works on macOS (Colima) and Linux/GitHub Actions.

**Key Design Decisions:**
- Use Testcontainers for database isolation
- Mock Ollama embedding server (deterministic, fast)
- Ryuk reaper for automatic cleanup
- Build tags: `//go:build integration` to separate slow integration tests
- Package-level container sharing (not per-test) for acceptable speed

---

## Work Units

### Work Unit 1: Add Testcontainers Dependencies

**Goal:** Add Testcontainers Go modules to the project.

**Files:**
- `go.mod`
- `go.sum`

**Steps:**
1. Add testcontainers dependencies:
   ```bash
   go get github.com/testcontainers/testcontainers-go@v0.35.0
   go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.35.0
   ```
2. Run `go mod tidy` to update go.sum

**Verification:**
- `go.mod` contains testcontainers entries
- `go build ./...` succeeds

---

### Work Unit 2: Create Test Utilities Package

**Goal:** Create shared test infrastructure for integration tests.

**Files:**
- `internal/testutil/db.go` (new)
- `internal/testutil/migrate.go` (new)

**Requirements:**
- `PostgresContainer` struct wrapping testcontainers
- `NewPostgresContainer(ctx)` - starts pgvector container with:
  - Image: `pgvector/pgvector:pg17`
  - Database: `trindex_test`
  - Wait strategy: Log-based ("database system is ready")
  - Lifespan: 10 minutes (safety)
- `SkipIfNoDocker(t)` - skips if Docker unavailable
- `IsCI()` - detects CI environment
- `SetupTestDB(ctx, connStr)` - creates pool, runs migrations
- `TruncateTables(ctx, pool)` - fast cleanup between tests

**Verification:**
- File compiles: `go build ./internal/testutil/...`
- Linter passes: `golangci-lint run ./internal/testutil/...`

---

### Work Unit 3: Create Mock Ollama Server

**Goal:** Deterministic mock embedding server for tests.

**Files:**
- `internal/testutil/mock_ollama.go` (new)

**Requirements:**
- `MockOllamaServer(embeddingDim int) *httptest.Server`
- Returns embeddings matching requested dimension
- Deterministic values (e.g., `[i*0.01 for i in range(dim)]`)
- OpenAI-compatible `/v1/embeddings` endpoint format

**Verification:**
- File compiles
- Unit test: `TestMockOllamaServer` that verifies response format

---

### Work Unit 4: Update Taskfile for Integration Tests

**Goal:** Add integration test tasks to build system.

**Files:**
- `Taskfile.yml`

**Requirements:**
Add tasks:
- `test:integration` - runs `go test -v -tags=integration ./...`
- `test:integration:ci` - CI mode with `CI=true`, `TESTCONTAINERS_RYUK_DISABLED=false`
- `test:all` - runs both unit and integration tests
- `colima:check` (macOS only) - verifies Colima is running
- `test:integration:mac` (macOS only) - checks Colima before running

**Verification:**
- `task --list` shows new tasks
- `task test:integration` runs (skips if no Docker)

---

### Work Unit 5: Create GitHub Actions Workflow

**Goal:** CI pipeline for integration tests.

**Files:**
- `.github/workflows/integration.yml` (new)

**Requirements:**
- Trigger: push to `main`, PRs to `main`
- Uses `ubuntu-latest`
- Setup Go 1.26.0
- Install go-task
- Run `task test:integration:ci`
- Environment variables:
  - `TESTCONTAINERS_RYUK_DISABLED: "false"`
  - `TESTCONTAINERS_RYUK_CONNECTION_TIMEOUT: "5m"`
  - `DOCKER_HOST: unix:///var/run/docker.sock`

**Verification:**
- YAML validates
- Workflow appears in GitHub Actions tab

---

### Work Unit 6: Write First Integration Test

**Goal:** Prove the setup works with a real integration test.

**Files:**
- `internal/memory/store_integration_test.go` (new)

**Requirements:**
- Build tag: `//go:build integration` (first line)
- Tests:
  - `TestStore_Create_Integration` - create memory, verify embedding stored
  - `TestStore_Recall_HybridSearch_Integration` - create multiple memories, verify hybrid search returns ordered results
- Use `testutil.NewPostgresContainer()` and `testutil.SetupTestDB()`
- Use `t.Cleanup()` for container termination
- Use `testutil.TruncateTables()` between sub-tests if needed

**Verification:**
- `go test -v -tags=integration ./internal/memory/...` passes (with Docker)
- `go test -v -short ./...` skips integration test (without tag)

---

### Work Unit 7: Add Integration Tests for Embed Client

**Goal:** Test embedding client with mock Ollama server.

**Files:**
- `internal/embed/client_integration_test.go` (new)

**Requirements:**
- `TestClient_Embed_Integration` - embed text, verify dimension
- `TestClient_EmbedBatch_Integration` - batch embed multiple texts
- `TestClient_ValidateDimensions_Integration` - test dimension validation logic
- Start mock Ollama server per test using `testutil.MockOllamaServer()`

**Verification:**
- Tests pass with mock server
- Tests fail appropriately when server returns wrong dimensions

---

### Work Unit 8: Add Integration Tests for Import/Export

**Goal:** Test import/export with real database operations.

**Files:**
- `internal/memory/import_export_integration_test.go` (new)

**Requirements:**
- `TestStore_Export_Integration` - create memories, export, verify JSONL format
- `TestStore_Import_Integration` - import JSONL, verify data integrity
- `TestStore_Import_WithDuplicateDetection_Integration` - test skip-existing logic
- `TestStore_FindDuplicates_Integration` - test similarity-based duplicate finding
- `TestStore_MergeDuplicates_Integration` - test merge transaction

**Verification:**
- All tests pass with real Postgres
- Proper cleanup between tests

---

### Work Unit 9: Document Colima Setup for macOS

**Goal:** Document local development setup for Mac users.

**Files:**
- `docs/development.md` (update or create)
- `.colima/colima.yaml` (optional, for shared config)

**Requirements:**
Document:
- Installing Colima: `brew install colima`
- Starting with sufficient resources: `colima start --cpu 4 --memory 8 --disk 50`
- Setting `DOCKER_HOST`: `export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"`
- Verifying Docker works: `docker ps`
- Running integration tests: `task test:integration:mac`

**Verification:**
- Documentation is clear and accurate
- Commands copy-paste correctly

---

### Work Unit 10: Update AGENT.md Progress

**Goal:** Mark integration testing complete in project roadmap.

**Files:**
- `AGENT.md`

**Requirements:**
- Find Phase 1, section 1.6.4 "Final integration test"
- Update status to completed
- Reference `plans/integration_testing.md` for details
- Add pointer to `internal/testutil/` for test utilities

**Verification:**
- AGENT.md reflects current state
- Links work correctly

---

## Acceptance Criteria

- [ ] `go test -v -tags=integration ./...` passes locally (with Docker)
- [ ] `go test -v -short ./...` passes without Docker (skips integration)
- [ ] GitHub Actions integration workflow passes
- [ ] All integration tests use `//go:build integration` tag
- [ ] Ryuk cleanup confirmed working (no container leaks)
- [ ] Tests work on macOS with Colima
- [ ] Tests work on Linux/GitHub Actions
- [ ] Mock Ollama provides deterministic embeddings
- [ ] Documentation exists for local setup

---

## References

- [Testcontainers for Go](https://gopkg.in/testcontainers/testcontainers-go)
- [Testcontainers Postgres Module](https://gopkg.in/testcontainers/testcontainers-go/modules/postgres)
- [Ryuk Reaper Documentation](https://java.testcontainers.org/features/configuration/)
- [Colima Documentation](https://github.com/abiosoft/colima)
- [Go Build Tags](https://pkg.go.dev/go/build)
