package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/dbehnke/trindex/internal/config"
	"github.com/dbehnke/trindex/internal/eval"
)

func main() {
	modeFlag := flag.String("mode", "cognitive", "Evaluation mode: 'cognitive' (exact-match) or 'stress' (massive generative semantic noise)")
	usersFlag := flag.Int("users", 20, "Number of concurrent tenants/namespaces for stress testing")
	noiseFlag := flag.Int("noise", 500, "Number of random memories to generate per user in stress mode")
	flag.Parse()

	slog.Info("initializing Trindex Cognitive Evaluation Suite...")

	cfg, err := config.LoadWithPath("")
	if err != nil {
		slog.Error("failed to load base config", "error", err)
		os.Exit(1)
	}

	// Hot-wire evaluation-specific config overrides if desired here
	// Ensure we have Ollama accessible locally since this runs out of process from normal docker-compose

	evaluator := eval.NewEvaluator(cfg)
	ctx := context.Background()

	slog.Info("bootstrapping ephemeral container architecture")
	if err := evaluator.Start(ctx); err != nil {
		slog.Error("failed to start evaluation container", "error", err)
		cleanup(ctx, evaluator)
		os.Exit(1)
	}
	defer cleanup(ctx, evaluator)

	var dataset eval.AgentDataset
	if *modeFlag == "stress" {
		dataset = eval.GenerateSemanticNoiseDataset(*usersFlag, *noiseFlag)
	} else {
		dataset = eval.GenerateDeveloperPersonaDataset()
	}

	slog.Info("=========================================================")
	slog.Info(fmt.Sprintf("RUNNING EVALUATION: %s", dataset.Name))
	slog.Info(fmt.Sprintf("DESCRIPTION: %s", dataset.Description))
	slog.Info("=========================================================")

	res, err := evaluator.Evaluate(ctx, dataset)
	if err != nil {
		slog.Error("evaluation failed during execution", "error", err)
		os.Exit(1)
	}

	slog.Info("=========================================================")
	slog.Info("COGNITIVE EVALUATION RESULTS:")
	slog.Info("=========================================================")
	slog.Info(fmt.Sprintf("TOTAL QUERIES:   %d", res.QueriesExecuted))
	slog.Info(fmt.Sprintf("PERFECT RECALLS: %d", res.PerfectRecalls))
	slog.Info(fmt.Sprintf("PARTIAL RECALLS: %d", res.PartialRecalls))
	slog.Info(fmt.Sprintf("FAILED RECALLS:  %d", res.FailedRecalls))
	slog.Info(fmt.Sprintf("AVERAGE LATENCY: %dms", res.AverageLatencyMs))
	slog.Info("=========================================================")

	if res.FailedRecalls > 0 || res.PerfectRecalls < res.QueriesExecuted {
		slog.Warn("cognitive recall suite failed to achieve 100% precision", "failed", res.FailedRecalls)
		os.Exit(1)
	}

	slog.Info("evaluation completed successfully: 100% precision recall achieved.")
	os.Exit(0)
}

func cleanup(ctx context.Context, evaluator *eval.Evaluator) {
	if err := evaluator.Teardown(ctx); err != nil {
		slog.Error("failed to teardown environment properly", "error", err)
	}
}
