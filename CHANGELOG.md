# Changelog

All notable changes to this project will be documented in this file.

## [1.1.1] - 2026-04-13

### Fixed
- Protocol version negotiation: server now echoes client's version instead of hardcoded `2024-11-05`, ensuring compatibility with Claude Desktop and Claude Code (which use `2025-11-25`).
- Stdio transport: switched from HTTP-style header parsing (`Content-Length`) to proper MCP newline-delimited JSON per spec.
- `listChanged` capability corrected from `false` to `true`.

### Changed
- `InitializeResult` now includes `instructions` field with full tool catalog, enabling better tool discovery by AI models.

### Added
- Claude Desktop compatible configuration example in README.

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