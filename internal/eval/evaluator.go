package eval

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/db"
	"github.com/dbehnke/trindex/internal/embed"
	"github.com/dbehnke/trindex/internal/memory"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Evaluator manages the spinup of the test architecture and the execution of datasets
type Evaluator struct {
	cfg   *config.Config
	db    *db.DB
	embed *embed.Client
	store *memory.Store
	pgC   *postgres.PostgresContainer
}

// Result holds the findings of a cognitive evaluation run
type Result struct {
	DatasetName      string
	QueriesExecuted  int
	PerfectRecalls   int
	PartialRecalls   int
	FailedRecalls    int
	AverageLatencyMs int64
	TokensSavedCount int // Hypothetical count of unneeded tokens dropped by precision recall
}

// NewEvaluator creates a new testing context
func NewEvaluator(cfg *config.Config) *Evaluator {
	return &Evaluator{
		cfg: cfg,
	}
}

// Start spins up an ephemeral pgvector container and initializes the database connections.
func (e *Evaluator) Start(ctx context.Context) error {
	slog.Info("spinning up ephemeral pgvector container...")

	pgContainer, err := postgres.Run(ctx,
		"pgvector/pgvector:pg17",
		postgres.WithDatabase("trindex_eval"),
		postgres.WithUsername("trindex"),
		postgres.WithPassword("trindex"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return fmt.Errorf("failed to start pgvector container: %w", err)
	}
	e.pgC = pgContainer

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	// Hot-wire the configuration to point at our ephemeral database
	e.cfg.DatabaseURL = connStr
	e.cfg.DBMaxConns = 10
	e.cfg.DBMinConns = 2

	slog.Info("container ready, connecting to database...", "url", connStr)

	database, err := db.New(e.cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to ephemeral database: %w", err)
	}
	e.db = database

	slog.Info("running migrations...")
	if err := e.db.Migrate(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	e.embed = embed.NewClient(e.cfg)
	e.store = memory.NewStore(e.db, e.embed, e.cfg)

	return nil
}

// Teardown kills the ephemeral container
func (e *Evaluator) Teardown(ctx context.Context) error {
	if e.db != nil {
		e.db.Close()
	}
	if e.pgC != nil {
		slog.Info("terminating ephemeral pgvector container...")
		return e.pgC.Terminate(ctx)
	}
	return nil
}

// Evaluate runs a simulated dataset session against the ephemeral system
func (e *Evaluator) Evaluate(ctx context.Context, dataset AgentDataset) (Result, error) {
	slog.Info("starting cognitive evaluation", "dataset", dataset.Name)

	result := Result{
		DatasetName: dataset.Name,
	}

	// 1. Inject memories sequentially to simulate chronological brain growth
	slog.Info("injecting simulated memories...", "count", len(dataset.Memories))
	for i, mem := range dataset.Memories {
		meta := make(map[string]interface{})
		for k, v := range mem.Metadata {
			meta[k] = v
		}

		ns := "default"
		if nsVal, ok := mem.Metadata["namespace"]; ok {
			ns = nsVal
		}

		// Embedding and insertion
		_, err := e.store.Create(ctx, mem.Content, ns, meta)
		if err != nil {
			return result, fmt.Errorf("failed to insert memory %d: %w", i, err)
		}

		if (i+1)%250 == 0 {
			slog.Info(fmt.Sprintf("... still injecting: %d / %d memories inserted", i+1, len(dataset.Memories)))
		}

		// Wait a tiny bit just to avoid tight-loop hammering the Ollama API too hard
		// In a real high-throughput load test, we'd fire these off concurrently
		time.Sleep(10 * time.Millisecond)
	}

	// 2. Run Queries
	slog.Info("executing cognitive recall queries...", "count", len(dataset.Queries))
	var totalLatency time.Duration

	for _, query := range dataset.Queries {
		start := time.Now()

		recallReq := memory.RecallParams{
			Query:     query.Query,
			TopK:      query.TopK,
			Threshold: 0.0001, // Bypass RRF artificial score deflation limit
		}

		if query.ContextHint != "" {
			recallReq.Namespaces = []string{query.ContextHint}
		} else {
			recallReq.Namespaces = []string{"default"}
		}

		resp, err := e.store.Recall(ctx, recallReq)
		if err != nil {
			slog.Error("failed to execute recall query", "query", query.Query, "error", err)
			continue
		}
		totalLatency += time.Since(start)
		result.QueriesExecuted++

		// Scoring
		// Extract actual content retrieved to match against expected strings
		slog.Info("query results fetched", "query", query.Query, "results_count", len(resp))
		actualMatches := make([]string, len(resp))
		for i, res := range resp {
			slog.Info("retrieved memory", "score", res.Score, "content", res.Content)
			actualMatches[i] = res.Content
		}

		matchCount := 0
		for _, exp := range query.Expected {
			found := false
			for _, act := range actualMatches {
				if strings.Contains(act, exp) {
					found = true
					matchCount++
					break
				}
			}
			if !found {
				slog.Warn("missed expected recall target", "query", query.Query, "expected", exp)
			}
		}

		if matchCount == len(query.Expected) {
			result.PerfectRecalls++
		} else if matchCount > 0 {
			result.PartialRecalls++
		} else {
			result.FailedRecalls++
		}
	}

	if result.QueriesExecuted > 0 {
		result.AverageLatencyMs = totalLatency.Milliseconds() / int64(result.QueriesExecuted)
	}

	return result, nil
}
