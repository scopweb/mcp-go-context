# MCP Go Context Server

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

MCP server written in Go for local project analysis, conversation memory, and documentation lookup.

The project is intended for editor and assistant workflows where it is useful to keep technical context close to the codebase instead of depending only on external services.

## What It Does

- analyzes the current project structure and dependencies
- stores and retrieves persistent memory between sessions
- fetches documentation with local fallbacks when possible
- exposes MCP tools over `stdio`, `http`, or `sse`
- serves a local memory dashboard when running over `http` or `sse`

## Install

You can build the binary directly:

```bash
git clone https://github.com/scopweb/mcp-go-context
cd mcp-go-context
go build -o bin/mcp-context-server ./cmd/mcp-context-server
```

Or use the Makefile:

```bash
make build
```

## Basic Usage

Run with the default `stdio` transport:

```bash
./bin/mcp-context-server --transport stdio
```

Run over HTTP:

```bash
./bin/mcp-context-server --transport http --port 3000
```

Show build version:

```bash
./bin/mcp-context-server --version
```

## Configuration

The server reads `config.json` when provided with `--config`, or falls back to built-in defaults.

Minimal example:

```json
{
  "transport": {
    "type": "stdio",
    "port": 3000
  },
  "memory": {
    "enabled": true,
    "persistent": true,
    "storagePath": "$HOME/.mcp-go-context/memory"
  }
}
```

## Editor Integration

Claude Desktop example:

```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "/absolute/path/to/bin/mcp-context-server",
      "args": ["--transport", "stdio"]
    }
  }
}
```

Cursor example:

```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "/absolute/path/to/bin/mcp-context-server",
      "args": ["--transport", "stdio"]
    }
  }
}
```

## Dashboard

The memory dashboard is available only when the server runs with `http` or `sse` transport.

Example:

```json
{
  "transport": {
    "type": "http",
    "port": 3000
  }
}
```

With that configuration:

- `http://localhost:3000/dashboard` serves the web UI
- `http://localhost:3000/api/memories` serves the JSON API

The dashboard is intended for local inspection and maintenance of stored memories: search, filtering, read-only inspection, and deletion.

## MCP Tools

`analyze-project`
Analyzes the project structure, relevant files, and dependencies.

`get-context`
Builds contextual output for a query using project analysis and stored memory.

`fetch-docs`
Fetches documentation for a library or topic, with fallbacks when remote lookup is unavailable.

`remember-conversation`
Stores conversation context with tags for later retrieval.

`dependency-analysis`
Extracts dependency information and related recommendations.

## Development

Run tests:

```bash
go test ./...
```

Run the HTTP transport locally:

```bash
make run-http
```

Run the SSE transport locally:

```bash
make run-sse
```

## License

MIT. See [LICENSE](LICENSE).
