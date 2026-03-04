# Speaker Script: Trindex Presentation

## Slide 1: Title
**Duration:** 30 seconds

"Hi everyone, today I'm going to talk about Trindex - a project I've been working on to solve a fundamental problem in AI agent development.

Trindex is persistent semantic memory for AI agents. It's a standalone Go binary that lets any MCP-compatible agent remember things across conversations."

**Key points:**
- Standalone Go binary
- Language-agnostic via MCP
- Persistent memory

---

## Slide 2: What is Trindex?
**Duration:** 2 minutes

"Let me break down what Trindex actually does.

First, it stores memories as vector embeddings in PostgreSQL using the pgvector extension. This means you get production-grade persistence with ACID compliance.

Second, it retrieves memories using semantic similarity - so if an agent searches for 'machine learning basics', it finds content about neural networks, deep learning, etc., even if those exact words weren't used.

Third, it uses hybrid search - combining vector similarity with traditional full-text search, fused using RRF - Reciprocal Rank Fusion. This gives you the best of both worlds.

Fourth, it supports namespaces - so you can have isolated memory spaces for different projects, but still search global memories.

And finally, it provides multiple interfaces: REST API for programmatic access, Web UI for browsing, and CLI for scripting."

**Key points:**
- Vector embeddings in Postgres
- Semantic similarity search
- Hybrid search with RRF
- Namespaces
- Multiple interfaces

---

## Slide 3: The Problem
**Duration:** 2 minutes

"Why did I build this? Because AI agents have no memory.

Every conversation starts fresh. If you're using Claude Code or any AI coding assistant, you know the pain of re-explaining your project structure, your conventions, your architecture decisions - every single session.

Context windows are limited. Even with 200k token windows, you can't fit an entire codebase or long conversation history.

And there's no standard way to persist knowledge across sessions. Each agent vendor has their own solution, or none at all.

I looked at existing solutions:
- LangChain is great but Python-only and brings heavy dependencies
- OpenBrain has the right architecture but requires manual setup
- Vector DBs like Pinecone are powerful but low-level - you still need to build the MCP integration

I wanted something standalone, language-agnostic, and MCP-native from the ground up. That's Trindex."

**Key points:**
- Agents start fresh every session
- Limited context windows
- No persistence standard
- Existing solutions are language-specific or low-level

---

## Slide 4: Key Features
**Duration:** 2 minutes

"Trindex has three key features that make it unique.

First, it's MCP-native. The Model Context Protocol is an open standard from Anthropic that lets AI agents discover and use tools. Trindex exposes memory tools via MCP, so any compatible client can use it immediately. The configuration is simple - just point to the binary.

Second, semantic search. When you store a memory like 'The user prefers React functional components with hooks', and later search for 'frontend framework preferences', Trindex finds it. It uses cosine similarity via pgvector, combined with PostgreSQL's full-text search, merged with RRF fusion for optimal results.

Third, namespace organization. You can create isolated memory spaces - one for 'work-project', one for 'personal', one for 'research'. But the global namespace is always included in searches, so common knowledge is shared while project-specific details are isolated."

**Key points:**
- MCP-native integration
- Semantic + full-text hybrid search
- Namespace isolation with global fallback

---

## Slide 5: Architecture
**Duration:** 2 minutes

"Here's how Trindex fits together.

At the top, you have MCP clients - Claude Code, opencode, Cursor, or any custom orchestrator. They communicate with Trindex via stdio - the MCP standard.

Trindex itself has two modes: 'mcp' mode for agent integration, and 'server' mode for HTTP API and Web UI. This separation is key - you can run just the HTTP server for centralized deployments, or just MCP for local agent usage.

The REST API provides CRUD operations for memories, search endpoints, stats, and import/export functionality.

Underneath, everything stores in PostgreSQL with the pgvector extension. This gives you ACID transactions, backups, replication - all the production database features you need.

The embedding service is pluggable - Ollama for local development, OpenAI for production, or any OpenAI-compatible endpoint."

**Key points:**
- MCP clients talk stdio
- Two modes: mcp and server
- REST API for programmatic access
- PostgreSQL + pgvector backend
- Pluggable embeddings

---

## Slide 6: CLI Redesign
**Duration:** 3 minutes

"The recent CLI redesign is what I want to highlight. This was a major usability improvement.

Before, Trindex was monolithic. You ran './trindex' and it started everything: MCP server, HTTP server, database connection. This had problems: you couldn't run just the HTTP server for a centralized deployment, there was no way to access the REST API from command line without curl, and there were no built-in diagnostics.

The new CLI uses explicit subcommands:
- 'mcp' starts just the MCP server
- 'server' starts just the HTTP server
- 'doctor' runs diagnostics - checks config, database, embedding endpoint
- 'memories list/get/create/delete' gives you CLI access to the REST API
- 'search' lets you search from the command line
- 'export/import' for backups

This enables new workflows. You can run 'trindex doctor' to verify your setup before starting services. You can script memory operations. You can run the HTTP server on a central instance while agents connect via MCP proxy mode."

**Key points:**
- Old: monolithic, everything at once
- New: explicit subcommands
- Enables standalone deployment
- CLI access to REST API
- Diagnostics command

---

## Slide 7: CLI Demo Commands
**Duration:** 3 minutes (with live demo or explanation)

"Let me show you the CLI in action.

First, diagnostics. Run './trindex doctor' and it checks your configuration, tests database connectivity, and validates the embedding endpoint. You get clear pass/fail indicators with helpful error messages.

For server management, './trindex server --port 3000' starts just the HTTP server. './trindex mcp' starts just the MCP server.

For memory operations: './trindex memories list --namespace work --json' lists memories with JSON output. './trindex memories create' with content, namespace, and metadata flags creates a memory.

The search command: './trindex search "architecture patterns" --namespace work --top-k 10' performs semantic search and returns ranked results.

These commands use the REST API under the hood, so they're communicating with the server - either local or remote via --api-url."

**Key points:**
- Doctor command for setup verification
- Separate mcp and server commands
- Full CRUD via CLI
- Search from command line
- Uses REST API

---

## Slide 8: Technical Stack
**Duration:** 1 minute

"Quick overview of the technical stack.

Go 1.26+ for the implementation - fast, single binary, easy deployment.

PostgreSQL 17 with pgvector for storage - production-grade, supports vector similarity search.

HNSW index for fast approximate nearest neighbor search with cosine distance.

Hybrid search combining pgvector for vectors and PostgreSQL's tsvector for full-text, merged with RRF.

OpenAI-compatible embedding API - use Ollama locally, OpenAI in production, or LM Studio.

Chi router for HTTP, Vue 3 with Tailwind for the web UI, and testcontainers-go for integration testing with real PostgreSQL."

**Key points:**
- Go, PostgreSQL, pgvector
- HNSW index
- Hybrid search
- Pluggable embeddings
- Modern web stack

---

## Slide 9: Database Schema
**Duration:** 1 minute

"The database schema is straightforward.

We have an ID, namespace, content, and the vector embedding. Metadata is JSONB for flexibility - store any structured data. The search_vec is a generated tsvector column for full-text search.

The key index is the HNSW index on the embedding column using vector_cosine_ops. This gives you fast approximate nearest neighbor search.

The schema is designed to be simple but powerful. You can query it directly with SQL if needed, which is great for analytics or debugging."

**Key points:**
- Simple schema
- Generated tsvector column
- HNSW index for vector search
- Direct SQL access

---

## Slide 10: Why Trindex?
**Duration:** 1 minute

"Why should you use Trindex?

For AI agent developers, it's drop-in MCP memory. No Python dependencies, works with any MCP client, not just LangChain.

For DevOps, it's a single binary with Docker Compose support. Uses PostgreSQL, which you probably already run.

For end users, there's a web UI for browsing, CLI for scripting, and import/export for backups. You're not locked in."

**Key points:**
- Drop-in MCP memory
- Single binary deployment
- Web UI + CLI
- No vendor lock-in

---

## Slide 11: Future Roadmap
**Duration:** 1 minute

"Future plans include enterprise features like authentication, RBAC, multi-tenancy, and audit logging.

Advanced search capabilities: reranking with cross-encoders, query expansion, automatic namespace detection based on context, and memory decay - forgetting old, unused memories.

And ecosystem growth: LangChain integration for the Python folks, a native Python client, webhook support for triggering actions, and memory sharing between agents."

**Key points:**
- Enterprise features
- Advanced search
- Multi-language support

---

## Slide 12: Getting Started
**Duration:** 1 minute

"Getting started is simple.

Clone the repo, copy .env.example to .env, and edit it with your embedding endpoint configuration.

Then either run with Docker Compose - that's the easiest way - or build locally with Go.

The 'trindex doctor' command will verify your setup before you start services."

**Key points:**
- Clone and configure
- Docker Compose or local build
- Doctor command verifies setup

---

## Slide 13: Demo Time
**Duration:** 3-5 minutes

"Let me show you Trindex in action. [Perform live demo or reference recorded demo]

First, I'll run diagnostics to verify everything is configured correctly.

Then start the server and create some memories via CLI.

Next, I'll search for them to show semantic retrieval.

Then show the Web UI for browsing.

Finally, demonstrate export and import for backups."

**Key points:**
- Live demo of key features
- Show diagnostics, create, search, UI, export

---

## Slide 14: Questions
**Duration:** 2-5 minutes

"Before we wrap up, any questions?

You can find the code on GitHub at github.com/dbehnke/trindex. The docs/cli.md file has complete CLI reference.

The MCP specification is at modelcontextprotocol.io if you want to build your own MCP tools.

The project is under Business Source License 1.1, which means it's free for non-production use."

**Key points:**
- Open for questions
- GitHub and docs links
- License information

---

## Slide 15: Thank You
**Duration:** 30 seconds

"Thank you! Trindex is about giving AI agents persistent memory - one brain for every agent.

The goal is to make AI agents truly useful by letting them remember what matters. I hope you'll check it out and let me know what you think."

**Key points:**
- One brain, every agent
- Check it out
- Feedback welcome

---

## Presentation Tips

### Total Duration: ~25-30 minutes

### Pacing:
- Slides 1-5: 8-10 minutes (introduction and architecture)
- Slides 6-7: 5-6 minutes (CLI redesign - key highlight)
- Slides 8-11: 4-5 minutes (technical details)
- Slide 12: 1 minute (getting started)
- Slide 13: 3-5 minutes (demo)
- Slides 14-15: 3-5 minutes (Q&A and close)

### Emphasis:
- Spend extra time on the CLI redesign (slide 6) - this is the recent major improvement
- The demo (slide 13) is crucial - show don't just tell
- Highlight the problem (slide 3) to establish why this matters

### Potential Questions:
**Q: How does this compare to LangChain's memory?**
A: LangChain is Python-only and requires you to use their framework. Trindex is language-agnostic via MCP - any agent can use it.

**Q: What about scaling?**
A: PostgreSQL with pgvector scales well. HNSW indexes are efficient. For massive scale, you could shard by namespace.

**Q: How accurate is the semantic search?**
A: It depends on your embedding model. With 768-dim models like nomic-embed-text, it's quite good for technical content.

**Q: Can I use this with my existing Postgres?**
A: Yes, just enable the pgvector extension and run migrations.

**Q: What about security?**
A: Currently relies on network isolation. Future versions will add authentication and RBAC.
