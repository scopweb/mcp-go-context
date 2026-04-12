# Changelog

All notable changes to this project will be documented in this file.

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