# CLI Redesign Plan

> Restructure Trindex CLI with explicit subcommands for MCP, server, diagnostics, and REST operations.

## Overview

This plan redesigns the Trindex CLI from a single binary that starts everything (MCP + HTTP + DB) to an explicit subcommand-based interface. This enables better debugging, standalone HTTP server deployment, and future MCP proxy mode for centralized brain architecture.

## Current State

```bash
./trindex                      # Starts MCP (stdio) + HTTP (port 8080) + DB connection
```

**Problems:**
- No way to run HTTP server without MCP (for centralized deployment)
- No built-in diagnostic tools
- No CLI access to REST API without external tools (curl)
- Ambiguous what "./trindex" actually starts

## Proposed CLI Structure

### Primary Commands

```bash
# MCP Mode - Primary AI agent interface
trindex mcp [flags]
  --config PATH                # Config file path (default: ~/.config/trindex/config.yaml)
  
  # Future: Remote MCP proxy mode
  --remote URL                 # Remote Trindex HTTP API URL
  --api-key KEY                # API key for remote connection

# HTTP Server Mode - Standalone REST API server
trindex server [flags]
  --host HOST                  # HTTP host (default: 0.0.0.0)
  --port PORT                  # HTTP port (default: 8080)
  --no-ui                      # Disable web UI, API only

# Diagnostic Commands
trindex doctor                 # Check configuration and connectivity
trindex version                # Show version info
trindex config validate        # Validate config file

# REST API CLI Commands (direct HTTP API access)
trindex memories list [flags]
  --namespace NS               # Filter by namespace
  --limit N                    # Limit results (default: 20)
  --offset N                   # Pagination offset
  --order asc|desc             # Sort order

trindex memories get ID        # Get single memory by ID

trindex memories create [flags]
  --content "text"             # Memory content
  --namespace NS               # Namespace (default: default)
  --metadata key=value         # Metadata key-value pairs (repeatable)
  --file PATH                  # Read content from file
  --interactive                # Interactive mode (prompt for input)

trindex memories delete ID [flags]
  --force                      # Skip confirmation

trindex search "query" [flags]
  --namespace NS               # Search namespace (repeatable)
  --top-k N                    # Number of results (default: 10)
  --threshold 0.0-1.0          # Similarity threshold
  --json                       # Output as JSON

trindex stats [flags]
  --namespace NS               # Stats for specific namespace
  --json                       # Output as JSON

trindex export [flags]
  --output FILE                # Output file (default: stdout)
  --namespace NS               # Export specific namespace
  --since DATE                 # Export memories since date
  --until DATE                 # Export memories until date

trindex import FILE [flags]
  --skip-existing              # Skip duplicates
  --namespace NS               # Import to specific namespace
```

### Global Flags

```bash
--config PATH                  # Config file path
--env-file PATH                # .env file path
--log-level debug|info|warn|error
--json                         # Output as JSON (for scripting)
--api-key KEY                  # API key for REST commands
--api-url URL                  # Trindex HTTP API URL (for REST commands)
```

## Architecture Changes

### 1. Command Router

```go
// cmd/trindex/main.go becomes command router
func main() {
    if len(os.Args) < 2 {
        // Default to 'mcp' for backward compatibility during transition
        // Or show help
        fmt.Println("Usage: trindex <command> [flags]")
        fmt.Println("\nCommands:")
        fmt.Println("  mcp       Run MCP server (stdio)")
        fmt.Println("  server    Run HTTP server only")
        fmt.Println("  doctor    Run diagnostics")
        fmt.Println("  memories  Memory operations")
        fmt.Println("  search    Search memories")
        fmt.Println("  stats     Show statistics")
        fmt.Println("  export    Export memories")
        fmt.Println("  import    Import memories")
        os.Exit(1)
    }

    switch os.Args[1] {
    case "mcp":
        runMCP()
    case "server":
        runServer()
    case "doctor":
        runDoctor()
    case "memories":
        runMemoriesCommand(os.Args[2:])
    case "search":
        runSearch()
    case "stats":
        runStats()
    case "export":
        runExport()
    case "import":
        runImport()
    default:
        fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
        os.Exit(1)
    }
}
```

### 2. MCP Command Implementation

```go
// internal/cmd/mcp.go
func runMCP() {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    // Check for remote mode (future)
    if cfg.RemoteURL != "" {
        runMCPProxy(cfg)
        return
    }

    // Local mode - full server with DB
    database, err := db.New(cfg)
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer database.Close()

    if err := database.Migrate(ctx); err != nil {
        slog.Error("failed to run migrations", "error", err)
        os.Exit(1)
    }

    embedClient := embed.NewClient(cfg)
    if err := embedClient.ValidateDimensions(); err != nil {
        slog.Error("embedding validation failed", "error", err)
        os.Exit(1)
    }

    // Start MCP server only (no HTTP)
    mcpServer := mcp.NewServer(cfg, database, embedClient)
    mcpServer.RegisterTools()

    slog.Info("Trindex MCP server starting", "transport", "stdio")
    if err := mcpServer.Run(ctx); err != nil {
        slog.Error("mcp server error", "error", err)
        os.Exit(1)
    }
}
```

### 3. Server Command Implementation

```go
// internal/cmd/server.go
func runServer() {
    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    // Override with CLI flags
    if hostFlag != "" {
        cfg.HTTPHost = hostFlag
    }
    if portFlag != "" {
        cfg.HTTPPort = portFlag
    }

    database, err := db.New(cfg)
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer database.Close()

    if err := database.Migrate(ctx); err != nil {
        slog.Error("failed to run migrations", "error", err)
        os.Exit(1)
    }

    embedClient := embed.NewClient(cfg)
    if err := embedClient.ValidateDimensions(); err != nil {
        slog.Error("embedding validation failed", "error", err)
        os.Exit(1)
    }

    // Start HTTP server only (no MCP)
    webServer := web.NewServer(cfg, database, embedClient)

    slog.Info("Trindex HTTP server starting", 
        "addr", fmt.Sprintf("%s:%s", cfg.HTTPHost, cfg.HTTPPort))
    
    if err := webServer.Run(ctx); err != nil {
        slog.Error("web server error", "error", err)
        os.Exit(1)
    }
}
```

### 4. Doctor Command Implementation

```go
// internal/cmd/doctor.go
func runDoctor() {
    fmt.Println("🔍 Trindex Doctor")
    fmt.Println()

    allPassed := true

    // Check 1: Config loading
    fmt.Print("Checking configuration... ")
    cfg, err := config.Load()
    if err != nil {
        fmt.Printf("❌ FAILED\n   %v\n", err)
        allPassed = false
    } else {
        fmt.Println("✅ PASSED")
        fmt.Printf("   Database URL: %s\n", maskPassword(cfg.DatabaseURL))
        fmt.Printf("   Embed Model: %s\n", cfg.EmbedModel)
        fmt.Printf("   Embed Dimensions: %d\n", cfg.EmbedDimensions)
    }
    fmt.Println()

    // Check 2: Database connectivity
    if cfg != nil {
        fmt.Print("Checking database connection... ")
        db, err := db.New(cfg)
        if err != nil {
            fmt.Printf("❌ FAILED\n   %v\n", err)
            allPassed = false
        } else {
            if err := db.Health(ctx); err != nil {
                fmt.Printf("❌ FAILED\n   %v\n", err)
                allPassed = false
            } else {
                fmt.Println("✅ PASSED")
                
                // Check migrations
                var count int
                err := db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM memories").Scan(&count)
                if err != nil {
                    fmt.Printf("   ⚠️  Table check failed: %v\n", err)
                } else {
                    fmt.Printf("   Memories in database: %d\n", count)
                }
            }
            db.Close()
        }
        fmt.Println()
    }

    // Check 3: Embedding endpoint
    if cfg != nil {
        fmt.Print("Checking embedding endpoint... ")
        client := embed.NewClient(cfg)
        dims, err := client.TestConnection()
        if err != nil {
            fmt.Printf("❌ FAILED\n   %v\n", err)
            allPassed = false
        } else {
            fmt.Println("✅ PASSED")
            fmt.Printf("   Endpoint: %s\n", cfg.EmbedBaseURL)
            fmt.Printf("   Returned dimensions: %d\n", dims)
            if dims != cfg.EmbedDimensions {
                fmt.Printf("   ⚠️  WARNING: Configured dimensions (%d) != Actual dimensions (%d)\n",
                    cfg.EmbedDimensions, dims)
            }
        }
        fmt.Println()
    }

    // Check 4: File permissions
    fmt.Print("Checking data directory... ")
    // ...

    if allPassed {
        fmt.Println("🎉 All checks passed! Trindex is ready to go.")
        os.Exit(0)
    } else {
        fmt.Println("❌ Some checks failed. Please fix the issues above.")
        os.Exit(1)
    }
}
```

### 5. REST CLI Commands

```go
// internal/cmd/memories.go
func runMemoriesList() {
    cfg := loadCLIConfig()
    
    // Build query params
    params := url.Values{}
    if namespaceFlag != "" {
        params.Set("namespace", namespaceFlag)
    }
    params.Set("limit", strconv.Itoa(limitFlag))
    params.Set("offset", strconv.Itoa(offsetFlag))

    // Make HTTP request
    url := fmt.Sprintf("%s/api/memories?%s", cfg.APIURL, params.Encode())
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("X-API-Key", cfg.APIKey)

    resp, err := httpClient.Do(req)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        body, _ := io.ReadAll(resp.Body)
        fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
        os.Exit(1)
    }

    var memories []memory.Memory
    json.NewDecoder(resp.Body).Decode(&memories)

    // Output
    if jsonOutput {
        // JSON output
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        enc.Encode(memories)
    } else {
        // Table output
        w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
        fmt.Fprintln(w, "ID\tNAMESPACE\tCONTENT\tCREATED")
        for _, m := range memories {
            content := m.Content
            if len(content) > 50 {
                content = content[:47] + "..."
            }
            fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", 
                m.ID, m.Namespace, content, m.CreatedAt.Format("2006-01-02"))
        }
        w.Flush()
    }
}
```

## Configuration File Support

Add YAML config file support for easier management:

```yaml
# ~/.config/trindex/config.yaml
database:
  url: postgres://trindex:trindex@localhost:5432/trindex?sslmode=disable

embedding:
  base_url: http://localhost:11434/v1
  model: nomic-embed-text
  api_key: ollama
  dimensions: 768

server:
  host: 0.0.0.0
  port: 8080
  api_key: ""  # Set for production

recall:
  default_namespace: default
  default_top_k: 10
  default_threshold: 0.7

# For client/proxy mode (future)
remote:
  url: https://brain.example.com
  api_key: ${TRINDEX_API_KEY}
```

Environment variables still work and override file config.

## Implementation Phases (TDD Approach)

Each phase follows TDD: write tests first, then implementation.

### Phase 1: Command Router Foundation

**Test First:**
```go
// internal/cmd/router_test.go
func TestCommandRouter(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantCmd  string
        wantErr  bool
    }{
        {"no args shows help", []string{}, "", false},
        {"mcp command", []string{"mcp"}, "mcp", false},
        {"server command", []string{"server"}, "server", false},
        {"doctor command", []string{"doctor"}, "doctor", false},
        {"unknown command errors", []string{"foo"}, "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd, err := ParseCommand(tt.args)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.wantCmd, cmd.Name)
            }
        })
    }
}

func TestHelpOutput(t *testing.T) {
    output := captureOutput(func() {
        Run([]string{})
    })
    
    assert.Contains(t, output, "Usage:")
    assert.Contains(t, output, "mcp")
    assert.Contains(t, output, "server")
    assert.Contains(t, output, "doctor")
}
```

**Then Implement:**
- Command router that shows help on no args
- Routes to appropriate command handler
- Error on unknown commands

### Phase 2: MCP Command

**Test First:**
```go
// internal/cmd/mcp_test.go
func TestMCPCommand(t *testing.T) {
    t.Run("requires database connection", func(t *testing.T) {
        // Given invalid database URL
        os.Setenv("DATABASE_URL", "postgres://invalid:5432/db")
        
        // When running mcp command
        err := runMCP(context.Background())
        
        // Then should fail with connection error
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "database")
    })
    
    t.Run("validates embedding dimensions", func(t *testing.T) {
        // Given mock embedding server that returns wrong dims
        mockServer := testutil.MockOllamaServer(512) // Returns 512
        defer mockServer.Close()
        
        os.Setenv("EMBED_DIMENSIONS", "768") // Expects 768
        os.Setenv("EMBED_BASE_URL", mockServer.URL)
        
        // When running mcp command
        err := runMCP(context.Background())
        
        // Then should fail with dimension mismatch
        assert.Error(t, err)
    })
    
    t.Run("starts mcp server on stdio", func(t *testing.T) {
        // Given valid config with testcontainers db
        ctx := setupTestDB(t)
        
        // When running mcp command (with timeout)
        ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
        defer cancel()
        
        err := runMCP(ctx)
        
        // Then should start without error (timeout is expected)
        assert.Equal(t, context.DeadlineExceeded, err)
    })
}
```

**Then Implement:**
- MCP command that loads config
- Validates database connection
- Validates embedding endpoint
- Starts MCP server on stdio

### Phase 3: Server Command

**Test First:**
```go
// internal/cmd/server_test.go
func TestServerCommand(t *testing.T) {
    t.Run("starts http server on default port", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        defer cancel()
        
        err := runServer(ctx)
        
        // Should start server (timeout expected)
        assert.Equal(t, context.DeadlineExceeded, err)
    })
    
    t.Run("respects --host and --port flags", func(t *testing.T) {
        // Given custom host/port flags
        hostFlag = "127.0.0.1"
        portFlag = "9999"
        
        // When checking server config
        cfg := buildServerConfig()
        
        // Then should use custom values
        assert.Equal(t, "127.0.0.1", cfg.HTTPHost)
        assert.Equal(t, "9999", cfg.HTTPPort)
    })
    
    t.Run("no-mcp mode", func(t *testing.T) {
        // Given server command
        // When running
        // Then MCP should NOT be started
        // (Verify no stdio listeners)
    })
}
```

**Then Implement:**
- Server command with host/port flags
- Starts only HTTP server (no MCP)
- No stdio interaction

### Phase 4: Doctor Command

**Test First:**
```go
// internal/cmd/doctor_test.go
func TestDoctorCommand(t *testing.T) {
    t.Run("reports config errors", func(t *testing.T) {
        // Given invalid config
        os.Setenv("EMBED_DIMENSIONS", "invalid")
        
        // When running doctor
        output := captureOutput(func() {
            runDoctor()
        })
        
        // Then should report config failure
        assert.Contains(t, output, "❌ FAILED")
        assert.Contains(t, output, "EMBED_DIMENSIONS")
    })
    
    t.Run("reports database connectivity", func(t *testing.T) {
        // Given test database
        ctx := setupTestDB(t)
        
        // When running doctor
        output := captureOutput(func() {
            runDoctorContext(ctx)
        })
        
        // Then should report success
        assert.Contains(t, output, "✅ PASSED")
        assert.Contains(t, output, "database")
    })
    
    t.Run("exit code 0 on success", func(t *testing.T) {
        // Given valid setup
        // When running doctor
        exitCode := runDoctorWithExitCode()
        
        // Then exit code should be 0
        assert.Equal(t, 0, exitCode)
    })
    
    t.Run("exit code 1 on failure", func(t *testing.T) {
        // Given invalid setup
        os.Setenv("DATABASE_URL", "invalid")
        
        // When running doctor
        exitCode := runDoctorWithExitCode()
        
        // Then exit code should be 1
        assert.Equal(t, 1, exitCode)
    })
}
```

**Then Implement:**
- Doctor command with all health checks
- Proper exit codes (0 = pass, 1 = fail)
- Formatted output with ✅/❌

### Phase 5: REST CLI Commands

**Test First:**
```go
// internal/cmd/memories_test.go
func TestMemoriesList(t *testing.T) {
    t.Run("queries api and outputs table", func(t *testing.T) {
        // Given mock HTTP server
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            assert.Equal(t, "/api/memories", r.URL.Path)
            assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
            
            json.NewEncoder(w).Encode([]memory.Memory{
                {ID: uuid.New(), Content: "Test memory", Namespace: "default"},
            })
        }))
        defer server.Close()
        
        // When running list command
        output := captureOutput(func() {
            runMemoriesList(server.URL, "test-key")
        })
        
        // Then should output table
        assert.Contains(t, output, "Test memory")
        assert.Contains(t, output, "default")
    })
    
    t.Run("json output flag", func(t *testing.T) {
        // Given mock server
        // When running with --json
        // Then should output valid JSON
    })
    
    t.Run("namespace filter", func(t *testing.T) {
        // Given mock server that validates query params
        // When running with --namespace foo
        // Then request should include namespace param
    })
}

func TestMemoriesCreate(t *testing.T) {
    t.Run("creates memory with content", func(t *testing.T) {
        // Given mock server
        // When running create --content "test"
        // Then should POST to /api/memories
    })
    
    t.Run("creates with metadata", func(t *testing.T) {
        // When running with --metadata key=value
        // Then metadata should be included in request
    })
}

func TestSearchCommand(t *testing.T) {
    t.Run("searches with query", func(t *testing.T) {
        // Given mock search endpoint
        // When running search "my query"
        // Then should POST to /api/search with query
    })
}

func TestExportCommand(t *testing.T) {
    t.Run("exports to file", func(t *testing.T) {
        // Given temp file path
        // When running export --output /tmp/test.jsonl
        // Then file should contain valid JSONL
    })
}
```

**Then Implement:**
- memories list/get/create/delete commands
- search command
- stats command
- export command
- import command

### Phase 6: Integration Tests

**E2E Tests:**
```go
// tests/e2e/cli_test.go
func TestCLIEndToEnd(t *testing.T) {
    t.Run("full workflow", func(t *testing.T) {
        // 1. Start server
        // 2. Run doctor (should pass)
        // 3. Create memory via CLI
        // 4. List memories (should show created)
        // 5. Search (should find memory)
        // 6. Export
        // 7. Import
        // 8. Stop server
    })
    
    t.Run("mcp mode", func(t *testing.T) {
        // 1. Start mcp mode
        // 2. Send MCP request via stdin
        // 3. Verify response on stdout
    })
}
```

## MCP Configuration

MCP clients **must** use explicit subcommand:

```json
// opencode.json
{
  "mcp": {
    "trindex": {
      "type": "local",
      "command": ["/path/to/trindex", "mcp"],
      "enabled": true
    }
  }
}
```

```bash
# Claude Code
claude mcp add trindex --command "/path/to/trindex mcp"

# Note: Old configuration without "mcp" subcommand will not work
```

## Testing Strategy (TDD)

### Test Organization

```
internal/cmd/
├── router.go           # Command routing logic
├── router_test.go      # Router unit tests
├── mcp.go             # MCP command
├── mcp_test.go        # MCP command tests (with testcontainers)
├── server.go          # Server command
├── server_test.go     # Server command tests
├── doctor.go          # Doctor command
├── doctor_test.go     # Doctor command tests
├── memories.go        # Memory CLI commands
├── memories_test.go   # Memory command tests (with httptest)
├── search.go          # Search command
├── search_test.go     # Search command tests
├── export.go          # Export command
├── export_test.go     # Export tests
└── import.go          # Import command
    └── import_test.go # Import tests

tests/e2e/
└── cli_test.go        # End-to-end integration tests
```

### Testing Patterns

**Unit Tests:**
- Mock external dependencies (HTTP, DB)
- Test error cases and edge cases
- Verify command-line flag parsing

**Integration Tests:**
- Use testcontainers for real PostgreSQL
- Use httptest for mock HTTP server
- Verify full command execution flow

**E2E Tests:**
- Build and run actual binary
- Test complete user workflows
- Verify exit codes and output

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/cmd/...

# With coverage
go test -cover ./internal/cmd/...

# Integration tests only
go test -v -tags=integration ./tests/e2e/...

# E2E tests
go test -v ./tests/e2e/...
```

### Test Data

Use test fixtures for consistent test data:
- `internal/cmd/testdata/` - Test config files, JSONL samples
- Mock responses in test files

## Documentation Updates

Update following documents:
- README.md - New CLI usage examples
- AGENT.md - Architecture changes
- docs/development.md - Development workflow
- Create docs/cli.md - Full CLI reference

## Implementation Timeline (TDD Approach)

| Phase | Tasks | Est. Time |
|-------|-------|-----------|
| **Phase 1** | Write router tests, implement command router | 1-2 hours |
| **Phase 2** | Write MCP tests, implement mcp command | 2-3 hours |
| **Phase 3** | Write server tests, implement server command | 1-2 hours |
| **Phase 4** | Write doctor tests, implement doctor command | 1-2 hours |
| **Phase 5** | Write CLI REST tests, implement REST commands | 4-6 hours |
| **Phase 6** | E2E tests, integration testing | 2-3 hours |
| **Phase 7** | Documentation, polish | 1-2 hours |

**Total: 12-20 hours** (includes comprehensive test coverage)

## Future Enhancements

- `trindex mcp --remote` - MCP proxy mode for centralized brain
- `trindex config init` - Interactive config setup
- `trindex config set key value` - CLI config management
- `trindex logs` - View/query application logs
- `trindex backup` - Database backup/restore
- `trindex migrate` - Manual migration control

## Acceptance Criteria

### Core Commands
- [ ] `./trindex` shows help with available commands (no default behavior)
- [ ] `./trindex mcp` starts MCP server (stdio) with database connection
- [ ] `./trindex server` starts HTTP server only (no MCP, no stdio)
- [ ] `./trindex doctor` validates configuration and connectivity, exits 0/1
- [ ] `./trindex version` shows version information

### Test Coverage
- [ ] Command router tests: 100% coverage of argument parsing
- [ ] MCP command tests: config validation, DB connection, embedding check
- [ ] Server command tests: HTTP startup, flag parsing, no MCP interference
- [ ] Doctor command tests: all health checks, exit codes, output format
- [ ] CLI REST tests: all CRUD operations, error handling, JSON output
- [ ] E2E tests: full workflow integration

### REST CLI Commands
- [ ] `./trindex memories list` queries REST API with table output
- [ ] `./trindex memories list --json` outputs valid JSON
- [ ] `./trindex memories get ID` retrieves single memory
- [ ] `./trindex memories create --content "text"` creates memory
- [ ] `./trindex memories delete ID` deletes with confirmation
- [ ] `./trindex search "query"` performs hybrid search
- [ ] `./trindex stats` shows database statistics
- [ ] `./trindex export --output file.jsonl` exports memories
- [ ] `./trindex import file.jsonl` imports memories

### Configuration
- [ ] Environment variables work for all commands
- [ ] `--api-url` and `--api-key` flags work for REST commands
- [ ] `--config` flag loads YAML config file

### Documentation
- [ ] README.md updated with new CLI usage
- [ ] AGENT.md updated with architecture changes
- [ ] docs/cli.md created with full command reference
- [ ] MCP configuration examples updated

---

**Next Steps:**

### Immediate (Phase 1)
1. Create `internal/cmd/` package structure
2. Write router tests (TestCommandRouter, TestHelpOutput)
3. Implement command router
4. Verify all tests pass

### Phase 2-3
5. Write MCP command tests
6. Implement MCP command with testcontainers
7. Write server command tests
8. Implement server command

### Phase 4-6
9. Implement doctor command with tests
10. Implement REST CLI commands with tests
11. Write E2E integration tests

### Final
12. Update all documentation
13. Update MCP configuration examples
14. Final review and merge

**No Decisions Required** - Pre-release, no backward compatibility concerns.
