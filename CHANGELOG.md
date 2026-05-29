# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added - Fase 0: Cold-Start Robustness for Context & Memory

- **Major improvement to `get-context` cold-start behavior** (addresses TECH-01):
  - New `EnsureLightIndex(maxFiles, maxDuration)` method performs bounded, git-aware project discovery on first use.
  - Git-changed files are now prioritized during light indexing and receive strong relevance boosts in `findRelevantFiles`.
  - `GetRelevantContext` (used by `get-context` tool and `/api/quick-context`) now automatically triggers light indexing when the analyzer cache is cold.
  - Added exported constants: `DefaultLightIndexMaxFiles`, `DefaultLightIndexMaxDuration`, `LightIndexCooldown`.
- New observability methods on `ProjectAnalyzer`:
  - `IsLightIndexed()`
  - `LightIndexStats()`
  - `ResetLightIndex()` (mainly for testing)
- `GetRecentlyChangedFiles` now feeds the internal recently-changed map used for ranking.
- Dedicated cold-start tests added in `analyzer_test.go`:
  - `TestEnsureLightIndexPopulatesCache`
  - `TestEnsureLightIndexRespectsMaxFiles`
  - `TestResetLightIndex`
  - `TestLightIndexStats`
- `/api/project-summary` now includes light index state (`indexed`, `cacheSize`, `recentlyChanged`, etc.) for observability.
- Updated `AnalyzerInterface` and test fakes to support new methods.
- Backward-compatible: `quickDiscovery()` now delegates to the new bounded implementation.

### Changed
- Relevance scoring in `findRelevantFiles` strengthened for recently modified and git-changed files.
- Light indexing is now time- and file-count bounded to keep first `get-context` calls responsive even on large repositories.

### Documentation
- Progress on TECH-01 (cold-start) and related memory usability improvements.

### Added - Fase 1: Intelligent Promotion Suggestions

- New `SuggestForPromotion(limit)` method in memory manager with transparent heuristic scoring based on:
  - Structured decisions (`decisionType` + `reason`)
  - Usage frequency
  - Recency
  - Decision-like language in content
- New MCP tool: `suggest-promotions` — returns the top candidates worth promoting, with ready-to-use `promote-memory` instructions.
- Tool is registered and available immediately alongside existing memory tools.
- Updated `MemoryInterface` and test fakes.

### Fase 3 - Dashboard & Promotion Observability

- New `GET /api/suggestions` endpoint that surfaces promotion candidates from the improved heuristic.
- `POST /api/memories/{key}` now supports promoting a memory directly from the dashboard.
- Dashboard UI now has a "Suggested for Promotion" section with one-click Promote buttons.
- Stats bar now shows "Promoted" count.
- Promoted memories are now tracked and visible in the web interface.

### Improved

- **`instructions` field in `initialize` response** (high impact for memory usage):
  - Now lists all 12 tools with clear descriptions.
  - Prominently documents the Memory Convergence model and recommended workflow:
    `save-decision` → `suggest-promotions` → `promote-memory`
  - Strongly guides models on when and how to use persistent memory tools.
  - This is one of the highest-leverage changes for making LLMs actually use the memory features effectively.

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