# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- New tool `memory-stats`: aggregate memory statistics (sessions, memories, promoted, decisions, usage hits, storage bytes, limits).
- `fetch-docs` accepts optional `libraryId` to skip the Context7 resolve step, plus `query` as alias for `topic` (closes TECH-02).
- `dependency-analysis` now tags every dependency with a scope (`root`, `package`, `app`, `service`, `module`) discovered via a bounded manifest walk (nested `apps/`, `services/`, etc.), and groups output by scope (closes TECH-03).
- Memories auto-link to each other (`related` field): storing a memory links it to up to 5 memories with shared tags (ranked by tag + word overlap), with bidirectional backlinks. `remember-conversation`, `save-decision` and `search-memory` surface the links. Deletes and evictions clean up dangling links.
- Light index self-healing: cached files are re-analyzed when their mtime changes, and any dependency manifest change (`go.mod`, `package.json`, `pyproject.toml`, `requirements.txt`) forces re-indexing even inside the cooldown window.

### Fixed
- `internal/memory` test build (duplicate `contains` helper).
- `InferProject` returned `"c"` for `C:\` on non-Windows hosts (`filepath.ToSlash` is a no-op there).

### Fase 0: Cold-Start Robustness
- Major improvements to `get-context` so it works well even without prior `analyze-project` call (addresses TECH-01).
- New `EnsureLightIndex()` with time and file count limits + git-aware prioritization.
- Better relevance scoring using recent git changes.
- New observability APIs and dedicated cold-start tests.

### Fase 1: Intelligent Memory Promotion
- New tool `suggest-promotions` that analyzes existing memories and recommends the best candidates for long-term storage.
- Significantly improved promotion heuristic (`SuggestForPromotion`) using decision structure, usage, recency, weighted keywords, and alternatives considered.
- Clear recommended workflow: `save-decision` → `suggest-promotions` → `promote-memory`.

### Fase 3: Dashboard & Promotion Observability
- New `GET /api/suggestions` endpoint.
- Ability to promote memories directly via `POST /api/memories/{key}`.
- Full "Suggested for Promotion" section in the web dashboard with one-click promote buttons.
- Promoted count in stats + "Promoted only" filter.
- Visual badges for promoted memories.

### Improved
- **`instructions` field** in the `initialize` response completely rewritten for much better tool discoverability and memory convergence guidance.
- Build scripts for Windows (`build.bat` and `build-enhanced.bat`) with clean output name `mcp-context-server.exe` in `bin/`.

### Documentation
- Major overhaul of the entire documentation site (`website/`).
- New guides: **Buenas Prácticas de Memoria** and **Dashboard y API HTTP**.
- Significantly improved **Flujo de Memoria** guide with real practical examples.
- Updated README with modern workflow and Windows build instructions.
- All phases (0, 1, 3) and new tools properly documented.

## [1.2.0] - 2026-05-20

### Changed
- Go version updated to 1.26.2
- README badge updated to reflect Go 1.26.2

### Added
- **Fase 5**: HTTP REST API endpoints for context enrichment:
  - `GET /api/project-summary` - lightweight project overview for session start
  - `GET /api/quick-context` - on-demand technical context for queries
- Dashboard handler now integrates with project analyzer for richer endpoints
- Memory convergence documentation in README explaining SessionMemory vs MCP memory roles

### Documentation
- README now includes:
  - HTTP REST API section with all endpoints and examples
  - Memory Convergence section explaining promotion flow
  - Transport Options (stdio, http, sse) with configuration examples
  - HTTP API examples for common use cases

## [1.1.1] - 2026-04-13

### Fixed
- Protocol version negotiation: server now echoes client's version instead of hardcoded `2024-11-05`, ensuring compatibility with Claude Desktop and Claude Code (which use `2025-11-25`).
- Stdio transport: switched from HTTP-style header parsing (`Content-Length`) to proper MCP newline-delimited JSON per spec.
- `listChanged` capability corrected from `false` to `true`.
- Stdio transport now auto-detects HTTP headers (`Content-Length`) vs LDJ mode, fixing connection timeout issues with Claude Desktop (which uses HTTP headers) while maintaining compatibility with Claude Code (LDJ mode).

### Changed
- `InitializeResult` now includes `instructions` field with full tool catalog, enabling better tool discovery by AI models.

### Added
- Claude Desktop compatible configuration example in README.
- **Fase 5**: Integrated `src/context` enrichment via MCP. Server now supports auto-detection of HTTP headers (Claude Desktop) vs LDJ mode (Claude Code), resolving 30s connection timeout issues.
- **Fase 6**: Memory convergence with SessionMemory. Added `promote-memory` and `get-promoted-memories` tools to mark high-value memories for long-term persistence. Memory items now have `promoted` (bool) and `confidence` (low/medium/high) fields. Session memory stays in summary.md; promoted memories persist in MCP memory.
- **Fase 7**: Documentation overhaul. README now includes workflow examples, tool table, architecture notes, and clear scope (what it is vs what it is NOT).

## [1.1.0] - 2026-04-12

### Added
- Separate memory dashboard module served over HTTP/SSE at `/dashboard`.
- JSON API for memory listing, filtering, retrieval, and deletion at `/api/memories`.
- Memory manager support for listing memories by recency and deleting entries by key.
- Regression tests for dashboard handlers and new memory manager operations.

### Changed
- HTTP and SSE transports can now register extra routes before startup.
- Server version metadata now uses centralized build information instead of duplicated hardcoded strings.
- Build scripts and Makefile now inject version and commit from a shared build info package.

### Fixed
- Memory usage updates are persisted correctly after retrieval.
- Memory cleanup now removes stale index entries consistently.
- Analyzer handling for `pyproject.toml` dependencies and git changed-files parsing is more robust.

### Documentation
- README documents how to access the dashboard and the API.

## [1.0.0] - 2026-04-12

### Added
- Initial production-ready MCP context server release with project analysis, memory, documentation fetch, and multi-transport support.