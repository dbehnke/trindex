package eval

import (
	"time"
)

// MemoryItem represents a piece of information injected into the Trindex memory over time.
type MemoryItem struct {
	Content   string            `json:"content"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
}

// EvaluationQuery represents a test case we will issue against Trindex to measure precision and recall.
type EvaluationQuery struct {
	Query       string   `json:"query"`
	ContextHint string   `json:"context_hint,omitempty"` // E.g., a specific namespace or user constraint
	TopK        int      `json:"top_k"`
	Expected    []string `json:"expected"` // Slice of exact string contents we EXPECT Trindex to recall in the Top K results
}

// AgentDataset represents a growing conversation or context window for an AI agent.
type AgentDataset struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Memories    []MemoryItem      `json:"memories"`
	Queries     []EvaluationQuery `json:"queries"`
}

// BaseTimeline creates a sequence of timestamps separated by 1 hour
func BaseTimeline(count int) []time.Time {
	now := time.Now().Add(-time.Duration(count) * time.Hour)
	timeline := make([]time.Time, count)
	for i := 0; i < count; i++ {
		timeline[i] = now.Add(time.Duration(i) * time.Hour)
	}
	return timeline
}

// GenerateDeveloperPersonaDataset returns a dataset simulating a Developer building a SaaS product over time.
func GenerateDeveloperPersonaDataset() AgentDataset {
	t := BaseTimeline(15)

	return AgentDataset{
		Name:        "Developer Persona",
		Description: "A simulated timeline of a software engineer changing their stack choices over the course of a project. Tests recency weighting and multi-hop fact retrieval.",
		Memories: []MemoryItem{
			{Content: "We are starting a new SaaS product called Trindex.", Timestamp: t[0], Metadata: map[string]string{"user": "dev"}},
			{Content: "For the frontend, we have decided to stick with standard React and Tailwind.", Timestamp: t[1], Metadata: map[string]string{"user": "dev"}},
			{Content: "The backend will be written in Python using FastAPI for quick iteration.", Timestamp: t[2], Metadata: map[string]string{"user": "dev"}},
			{Content: "We are storing all our data in MongoDB for flexibility.", Timestamp: t[3], Metadata: map[string]string{"user": "dev"}},
			{Content: "I found a great tutorial on using React Server Components, we will adopt that.", Timestamp: t[4], Metadata: map[string]string{"user": "dev"}},
			{Content: "The Python backend is getting too slow for our embeddings generation.", Timestamp: t[5], Metadata: map[string]string{"user": "dev"}},
			{Content: "We have decided to rewrite the backend in Go for better concurrency.", Timestamp: t[6], Metadata: map[string]string{"user": "dev"}},
			{Content: "MongoDB isn't handling vector search well. We need a Postgres database with pgvector.", Timestamp: t[7], Metadata: map[string]string{"user": "dev"}},
			{Content: "Just finished migrating the data from MongoDB to Postgres.", Timestamp: t[8], Metadata: map[string]string{"user": "dev"}},
			{Content: "React Server Components are causing too much overhead. We are going to rewrite the UI in Vue 3.", Timestamp: t[9], Metadata: map[string]string{"user": "dev"}},
			{Content: "Setup Trindex to use an embedded Ollama model for local LLM inference.", Timestamp: t[10], Metadata: map[string]string{"user": "dev"}},
			{Content: "I added a dark mode theme to the Vue app using a nice cyberpunk color palette.", Timestamp: t[11], Metadata: map[string]string{"user": "dev"}},
			{Content: "We need an API Key management system to gate the UI.", Timestamp: t[12], Metadata: map[string]string{"user": "dev"}},
			{Content: "I used chi router for the Go backend and added CORS restrictions.", Timestamp: t[13], Metadata: map[string]string{"user": "dev"}},
			{Content: "Just pushed the Trindex v1.0 code to GitHub.", Timestamp: t[14], Metadata: map[string]string{"user": "dev"}},
		},
		Queries: []EvaluationQuery{
			{
				// Tests recency override - they said React first, but Vue last.
				Query:    "Why are we rewriting the user interface?",
				TopK:     5,
				Expected: []string{"React Server Components are causing too much overhead. We are going to rewrite the UI in Vue 3."},
			},
			{
				// Tests chronological tracking.
				Query:    "Why did we rewrite the backend?",
				TopK:     5,
				Expected: []string{"We have decided to rewrite the backend in Go for better concurrency.", "The Python backend is getting too slow for our embeddings generation."},
			},
			{
				// Tests vector recall for specific technical choices.
				Query:    "What database is being used for vector search?",
				TopK:     3,
				Expected: []string{"MongoDB isn't handling vector search well. We need a Postgres database with pgvector."},
			},
			{
				// General context matching.
				Query:    "How does the local LLM work?",
				TopK:     2,
				Expected: []string{"Setup Trindex to use an embedded Ollama model for local LLM inference."},
			},
		},
	}
}
