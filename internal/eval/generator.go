package eval

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	actions    = []string{"deployed", "configured", "removed", "updated", "scaled", "monitored", "restarted", "optimized", "migrated", "audited"}
	systems    = []string{"the database", "the frontend", "the cache layer", "the load balancer", "the message queue", "the auth service", "the Kubernetes cluster", "the CI/CD pipeline", "the background workers"}
	reasons    = []string{"to improve performance.", "because it was failing.", "as part of the Q3 OKRs.", "due to a security vulnerability.", "to handle increased traffic.", "to reduce cloud costs.", "to comply with new regulations."}
	adjectives = []string{"critical", "routine", "emergency", "scheduled", "minor", "major", "experimental"}
)

// GenerateSemanticNoiseDataset creates a high-volume AgentDataset filled with randomized
// deterministic tech-ops jargon, spread across multiple namespaces. It injects specific
// Target Facts to ensure cognitive recall still works under heavily saturated vector conditions.
func GenerateSemanticNoiseDataset(namespaces int, memoriesPerNamespace int) AgentDataset {
	dataset := AgentDataset{
		Name:        fmt.Sprintf("Semantic Noise Stress Test (%d Namespaces, %d Total Vectors)", namespaces, namespaces*memoriesPerNamespace),
		Description: "Simulates a massive, multi-tenant conversational timeline filled with semantic noise to test database throughput, connection limits, and large-scale precision recall.",
		Memories:    make([]MemoryItem, 0, namespaces*memoriesPerNamespace),
		Queries:     make([]EvaluationQuery, 0),
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	baseTime := time.Now().Add(-time.Duration(namespaces*memoriesPerNamespace) * time.Minute)

	totalMemories := 0
	for ns := 1; ns <= namespaces; ns++ {
		namespaceName := fmt.Sprintf("tenant-%d", ns)

		for m := 0; m < memoriesPerNamespace; m++ {
			// Generate random syntactic noise
			adj := adjectives[r.Intn(len(adjectives))]
			action := actions[r.Intn(len(actions))]
			sys := systems[r.Intn(len(systems))]
			reason := reasons[r.Intn(len(reasons))]

			content := fmt.Sprintf("Completed %s maintenance: %s %s %s", adj, action, sys, reason)

			dataset.Memories = append(dataset.Memories, MemoryItem{
				Content:   content,
				Timestamp: baseTime.Add(time.Duration(totalMemories) * time.Minute),
				Metadata:  map[string]string{"type": "system_log", "priority": adj, "namespace": namespaceName},
			})
			totalMemories++
		}

		// Inject a highly specific "Target Fact" needle in this namespace's haystack
		targetSecret := fmt.Sprintf("Alpha-Bravo-%d-Niner", r.Intn(99999))
		targetSentence := fmt.Sprintf("The emergency override protocol code for the primary vault is %s.", targetSecret)

		dataset.Memories = append(dataset.Memories, MemoryItem{
			Content:   targetSentence,
			Timestamp: baseTime.Add(time.Duration(totalMemories) * time.Minute),
			Metadata:  map[string]string{"type": "secure_log", "namespace": namespaceName},
		})
		totalMemories++

		// Formulate a query specifically looking for that needle
		dataset.Queries = append(dataset.Queries, EvaluationQuery{
			Query:       "What is the emergency override protocol code for the primary vault?",
			ContextHint: namespaceName, // Restrict search to this specific tenant
			TopK:        3,             // RAG typical slicing
			Expected:    []string{targetSentence},
		})
	}

	return dataset
}
