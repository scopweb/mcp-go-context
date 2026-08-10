# MCP Go Context Server

[![Go](https://img.shields.io/badge/Go-1.26.2-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Version](https://img.shields.io/badge/Version-1.2.0-blue?style=flat-square)](CHANGELOG.md)

MCP server written in Go for local project analysis, conversation memory, and documentation lookup.

The project is intended for editor and assistant workflows where it is useful to keep technical context close to the codebase instead of depending only on external services.

## What It Does

- Analyzes project structure, dependencies, and multi-ecosystem workspaces (Go, Node.js, Python)
- Stores persistent memory between sessions (decisions, fixes, conventions) with automatic linking between related memories
- `get-context` works well even on cold start thanks to bounded git-aware discovery with a self-healing index (mtime + manifest aware)
- `suggest-promotions` helps discover high-value memories worth keeping long-term
- Fetches documentation via Context7 with local fallbacks
- Exposes MCP tools over `stdio`, `http`, or `sse`
- Serves a local memory dashboard when running over `http` or `sse`
- Provides HTTP REST API for lightweight context enrichment

## What It Is NOT

This is NOT a general-purpose memory system. It focuses on:

- Technical context for the current project
- Decision tracking and continuity
- Local documentation lookup

For session summarization and compaction, use SessionMemory (Claude Code's built-in feature).

## Quick Start

### 1. Build the binary

**Windows (recommended):**
```bat
git clone https://github.com/scopweb/mcp-go-context
cd mcp-go-context
build.bat
```

This will generate `bin\mcp-context-server.exe` (the file you configure in Claude Desktop).

**Linux / macOS:**
```bash
git clone https://github.com/scopweb/mcp-go-context
cd mcp-go-context
go build -o bin/mcp-context-server ./cmd/mcp-context-server
```

### 2. Configure Claude Desktop (Windows example):
```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "C:\\MCPs\\clone\\mcp-go-context\\bin\\mcp-context-server.exe",
      "args": ["--transport", "stdio", "--verbose"]
    }
  }
}
```

3. Restart Claude Desktop and start coding.

## Workflow Examples

### Onboarding to a New Repo

```
Use analyze-project to get a quick overview of the project structure and dependencies.
```

### Recovering Past Decisions

```
Use get-decisions with a keyword filter to find why a technical choice was made.
```

### Getting Context for a Task

```
Use get-context with a query like "database migrations" or "authentication flow".
```

### Tracking a Technical Decision (Memory Convergence)

```
1. Use save-decision with reason and alternatives when making important choices.
2. Before ending a complex session, call suggest-promotions to discover high-value items.
3. Use promote-memory on the best suggestions to preserve them long-term.
4. In future sessions, these will surface automatically via get-context and search-memory.
```

### Analyzing Dependencies

```
Use dependency-analysis to see all direct and indirect dependencies with recommendations.
```

### Fetching Library Documentation

```
Use fetch-docs with library name and optional topic. Falls back to local docs if offline.
```

### Enriching Context via HTTP API

When running with HTTP transport, external consumers can query context:

```bash
# Project summary for session start
curl "http://localhost:3000/api/project-summary?path=.&depth=2"

# On-demand technical context
curl "http://localhost:3000/api/quick-context?query=database+migrations&maxTokens=2000"
```

## MCP Tools

| Tool | Purpose |
|------|---------|
| `analyze-project` | Full project structure, dependencies, key files |
| `get-context` | Query-specific context from project + memory (works well even on cold start) |
| `dependency-analysis` | All dependencies with scope grouping (root/package/app/service/module) and recommendations; finds nested manifests in monorepos |
| `changed-files-context` | Context from recent git commits |
| `fetch-docs` | Documentation via Context7 (optional `libraryId` skips resolution) with local fallback |
| `remember-conversation` | Store any memory with tags (auto-links to related memories) |
| `save-decision` | Record a technical decision with type, reason and alternatives |
| `get-decisions` | Search decisions by type or keyword |
| `search-memory` | Full-text + ranked search across memories (shows related memories) |
| `suggest-promotions` | Analyzes memories and recommends which ones to promote to long-term storage |
| `promote-memory` | Mark a memory as high-value for persistent storage |
| `get-promoted-memories` | List only the high-value promoted memories |
| `memory-stats` | Aggregate memory statistics (counts, usage hits, storage size, limits) |

## HTTP REST API

When running with `http` or `sse` transport, these endpoints are available:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/dashboard` | GET | Web UI for memory management (now includes "Suggested for Promotion" with one-click promote) |
| `/api/memories` | GET | List/search memories |
| `/api/memories/{key}` | GET/DELETE/POST | Get, delete, or promote a specific memory |
| `/api/suggestions` | GET | Get promotion candidates from the smart heuristic (`suggest-promotions` logic) |
| `/api/project-summary` | GET | Lightweight project overview + memory stats (including promoted count) |
| `/api/quick-context` | GET | On-demand context for a technical query |

### HTTP API Examples

```bash
# List all memories
curl "http://localhost:3000/api/memories?limit=25"

# Search memories
curl "http://localhost:3000/api/memories?query=authentication&decisionType=architecture"

# Get project summary
curl "http://localhost:3000/api/project-summary?path=.&depth=2"

# Get technical context
curl "http://localhost:3000/api/quick-context?query=api+endpoint&maxTokens=3000"

# Get promotion suggestions (Fase 3)
curl "http://localhost:3000/api/suggestions?limit=5"

# Promote a memory from the dashboard/API
curl -X POST "http://localhost:3000/api/memories/my-important-decision"
```

When running the dashboard (`http` or `sse` transport), you will see a dedicated **"Suggested for Promotion"** section with the best candidates from the smart heuristic and one-click Promote buttons. This makes the memory convergence workflow (save-decision → suggest-promotions → promote) very practical from the browser.

## Memory Convergence

This server is designed to work **alongside** Claude Code's SessionMemory (summary.md):

| SessionMemory (summary.md) | mcp-go-context (this server) |
|---------------------------|-------------------------------|
| Transient session compaction | Persistent, high-value technical memory |
| Auto-generated summaries | Explicit decisions, fixes and conventions |
| Short-term context | Long-term retention across projects/sessions |

**Recommended workflow**:
1. Use `save-decision` when making architectural, security or important technical choices.
2. At the end of complex work, call `suggest-promotions` to surface the best candidates.
3. Use `promote-memory` to preserve the highest-value items.
4. Future sessions will automatically benefit from promoted memories via `get-context`.

The goal is selective, high-quality promotion — not dumping everything.

## Configuration

Default config (no file needed):

```json
{
  "context": {
    "projectPaths": ["."],
    "ignorePatterns": ["*.log", "*.tmp", "node_modules", ".git", "vendor"]
  },
  "memory": {
    "enabled": true,
    "persistent": true,
    "storagePath": "$HOME/.mcp-go-context/memory",
    "maxEntries": 1000
  },
  "transport": {
    "type": "stdio",
    "port": 3000
  }
}
```

### Transport Options

**stdio** (for Claude Desktop/Code):
```json
{ "transport": { "type": "stdio" } }
```

**http** (for web dashboard and external consumers):
```json
{ "transport": { "type": "http", "port": 3000 } }
```

**sse** (server-sent events for real-time updates):
```json
{ "transport": { "type": "sse", "port": 3000 } }
```

## Architecture Notes

- Memory convergence: Session memory (summary.md) handles session-level compaction. MCP memory handles persistent, high-value items.
- Use `promote-memory` to mark items that should survive beyond the current session.
- The server is stateless - all memory persists to disk.
- HTTP API provides lightweight enrichment without MCP overhead.

## Documentation

The project includes a full documentation site:

- **[Documentation Website](https://github.com/scopweb/mcp-go-context/tree/main/website)** (built with Astro + Starlight)
- [CHANGELOG](CHANGELOG.md)

**Recommended guides:**
- **Buenas Prácticas de Memoria** — How to use the memory system effectively
- **Flujo de Memoria** — Current recommended workflow using `suggest-promotions`
- **Dashboard y API HTTP** — How to use the web interface

## Development

```bash
# Run tests
go test ./...

# Build (Linux/macOS)
go build -o bin/mcp-context-server ./cmd/mcp-context-server

# Build (Windows - recommended)
build.bat

# Run with HTTP transport (for dashboard)
make run-http
```

## License

MIT. See [LICENSE](LICENSE).
