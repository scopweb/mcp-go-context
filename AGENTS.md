# AGENTS.md

MCP server in Go (stdio/http/sse) for project analysis, persistent memory, and doc lookup. Module: `github.com/scopweb/mcp-go-context`.

## Commands

- Build: `make build` (injects version/commit into `internal/buildinfo` via ldflags — bare `go build` produces an unversioned binary)
- Test all: `go test ./...` — single package: `go test ./internal/memory -run TestName -v`
- Format: `make fmt`; Lint: `make lint` (requires `golangci-lint` installed, no repo config file)
- Run: `make run-stdio` (default, for Claude Desktop), `make run-http` / `make run-sse` (port 3000, enables dashboard at `/dashboard` and REST API under `/api/`)
- No CI, no codegen, no external services needed for build or tests. Windows builds use `build-enhanced.bat` (README's `build.bat` reference is stale).

## Layout

- Entrypoint: `cmd/mcp-context-server/main.go`. Flags: `-transport`, `-port`, `-config`, `-verbose`, `-version`.
- `internal/server/server.go` — MCP protocol handling **and tool registration** (`registerTools`, ~line 270+). To add a tool: handler in `internal/tools/tools.go`, register in `server.go`, tests in `internal/tools/tools_test.go`.
- `internal/memory/manager.go` — memory persists as one JSON file per session under `$HOME/.mcp-go-context/memory` (configurable via `config.json` `memory.storagePath`). Project scope is a slug inferred from cwd at startup.
- `internal/analyzer`, `internal/dashboard` (web UI + REST), `internal/transport`, `internal/config`.
- `issues/` — design/tech-debt docs (TECH-*.md) that explain why features exist; check before changing memory or get-context behavior. `.claude/skills/mcp-go-context/SKILL.md` documents the tool surface.

## Gotchas

- **Zero external dependencies** — stdlib only. Do not add deps without strong reason; `go.mod` intentionally has no `require` block.
- `go.md` at repo root is a stale leftover, not a real file — ignore it.
- Tests are hermetic: memory tests use `t.TempDir()`, no network required (`fetch-docs` has local fallback).
- Docs drift: README and CONTRIBUTING occasionally lag the code — trust `Makefile`, `main.go` flags, and `server.go` registration over prose.
- Commits: conventional commits (`feat:`, `fix:`, ...).

## Continuity between clients

- At the start of substantive work, call `resume-context` with the repository path.
- Pass that same `path` to `save-decision` and `remember-conversation`.
- After a milestone or before switching clients, call `save-handoff` with `expectedRevision`.
- Files and Git remain the source of truth. A handoff is the shared work state, not a promoted decision.
- On `CONFLICT`, keep both versions and reconcile them.
- All local clients must use the same memory directory, by default `$HOME/.mcp-go-context/memory`.
