# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an MCP (Model Context Protocol) server written in Go that provides intelligent context management for AI coding assistants. **Version 2.0.1** includes MCP 2025-03-26 protocol support, Desktop Extensions (.dxt), JWT authentication, CORS security, Streamable HTTP transport, and performance optimizations.

### Recent Changes (v2.0.1)
- **Go 1.23**: Updated from 1.21 for latest optimizations
- **Performance**: Pre-compiled regexes (2-5ms faster), optimized allocations (20-30% reduction)
- **Organization**: Documentation moved to `/docs/` directory
- **Documentation**: New `CHANGELOG.md` and `docs/OPTIMIZATIONS.md`

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.

## Development Commands

### Build & Run
- **Build**: `go build -o bin/mcp-context-server.exe cmd/mcp-context-server/main.go` or use `build.bat` on Windows
- **Build Desktop Extension**: `./build-dxt.sh` or `build-dxt.bat` (creates .dxt package)
- **Run with stdio**: `./bin/mcp-context-server.exe --transport stdio --verbose`
- **Run with HTTP**: `./bin/mcp-context-server.exe --transport http --port 3000`
- **Run with SSE**: `./bin/mcp-context-server.exe --transport sse --port 3000`
- **Run with Streamable HTTP**: `./bin/mcp-context-server.exe --transport streamable-http --port 3000`

### Testing & Quality
- **Run all tests**: `go test -v ./...`
- **JWT tests**: `go test -v ./test -run TestJWTAuthSimple`
- **CORS tests**: `go test -v ./test -run TestCORS`
- **Streamable transport tests**: `go test -v ./test -run TestStreamable`
- **Test with coverage**: `go test -v -cover -coverprofile=coverage.out ./...`
- **Format code**: `go fmt ./...`
- **Lint** (requires golangci-lint): `golangci-lint run`

### Make Commands
The project includes a comprehensive Makefile with targets:
- `make build` - Build the binary
- `make test` - Run tests
- `make test-coverage` - Run tests with coverage
- `make fmt` - Format code  
- `make lint` - Lint code
- `make clean` - Clean build artifacts

## Architecture Overview

### Core Components

1. **Transport Layer** (`internal/transport/`): Handles multiple communication protocols
   - `stdio.go` - Standard input/output (Claude Desktop compatible)
   - `http.go` - HTTP JSON-RPC transport with CORS support
   - `sse.go` - Server-Sent Events transport with CORS support
   - `streamable.go` - **NEW**: Hybrid HTTP + SSE (MCP 2025-03-26)
   - Auto-detects protocol format for Claude Desktop compatibility

2. **Server** (`internal/server/server.go`): Main MCP server implementation
   - JSON-RPC 2.0 protocol handler (MCP 2025-03-26 compliant)
   - Tool registration and execution (11 tools available)
   - **NEW**: JWT authentication for HTTP/SSE transports
   - Graceful error handling and security logging

3. **Tools System** (`internal/tools/`): Extensible tool registry
   - `registry.go` - Tool registration and management
   - `tools.go` - Core tool implementations (analyze-project, get-context, etc.)
   - Interface-based design for easy extension

4. **Memory Management** (`internal/memory/manager.go`): Persistent conversation memory
   - Session-based storage with LRU eviction
   - JSON file persistence
   - Search and tagging capabilities
   - Thread-safe operations

5. **Project Analysis** (`internal/analyzer/analyzer.go`): Deep codebase understanding
   - AST parsing and analysis
   - Dependency mapping
   - File structure analysis
   - Context extraction

6. **Configuration** (`internal/config/config.go`): Enhanced JSON-based configuration system
7. **Security** (`internal/security/cors.go`): **NEW**: CORS middleware with configurable origins  
8. **Authentication** (`internal/auth/jwt.go`): **NEW**: JWT token management and validation

### MCP Tools Available (11 Total)

**Analysis Tools:**
- `analyze-project` - Comprehensive project analysis with metrics and dependency mapping
- `dependency-analysis` - Project dependency analysis with security recommendations
- `config-get-project-paths` - Get configured project paths

**Context & Documentation:**
- `get-context` - Intelligent context retrieval with memory integration
- `fetch-docs` - Documentation fetching with Context7 API integration and fallbacks

**Memory Management:**
- `remember-conversation` - Store important context for future reference with tagging
- `memory-get` - Retrieve a memory item by key
- `memory-search` - Search memories by text and/or tags with result limits
- `memory-recent` - Get recent memories (configurable limit)
- `memory-clear` - Clear all memories (requires explicit confirmation)

**Security & Development:**
- `auth-generate-token` - **NEW**: Generate JWT tokens for development/testing

### Key Architecture Decisions

1. **Transport Abstraction**: Single server handles stdio, HTTP, SSE, and **NEW** Streamable HTTP
2. **Interface-Based Design**: Tools, memory, and analyzers use interfaces for testability  
3. **Security-First**: JWT authentication, CORS protection, input validation, path sanitization
4. **MCP 2025 Compliance**: Full MCP 2025-03-26 protocol support with enhanced capabilities
5. **Memory Persistence**: File-based storage with session management and LRU eviction
6. **Desktop Extensions**: **NEW**: .dxt package support for one-click Claude Desktop installation

### Authentication & Security (HTTP/SSE only)

**JWT Authentication (Recommended):**
- Set `MCP_JWT_SECRET` environment variable to enable JWT auth
- Generate tokens: Use `auth-generate-token` tool  
- Include in requests: `Authorization: Bearer <jwt-token>`
- Auto-expiration: Configurable (default 1 hour)

**CORS Protection:**
- Configurable origin whitelist (no more wildcard `*`)
- Claude Desktop: `app://claude-desktop` automatically allowed
- Supports wildcard patterns: `*.yourdomain.com`

**Legacy Token Auth (Deprecated):**
- Set `MCP_SERVER_TOKEN` for simple token authentication
- Will be removed in future versions

### Entry Point

Main server starts in `cmd/mcp-context-server/main.go` with command-line flag parsing for transport type, port, and verbosity settings.

### Desktop Extensions (.dxt)

**Structure:**
- `dxt/manifest.json` - Extension metadata and configuration
- `dxt/README.md` - User documentation
- `dxt/package.json` - NPM-style package information  
- `build-dxt.bat/.sh` - Build scripts for creating .dxt packages

**Installation:**
1. Build: `./build-dxt.sh` or `build-dxt.bat`  
2. Distribute: `mcp-go-context.dxt` file
3. Install: Drag & drop into Claude Desktop
4. Configure: Optional JWT secrets and project paths via UI

### Claude Desktop Compatibility

**Traditional Configuration (Unchanged):**
```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "C:\\path\\to\\mcp-context-server.exe",
      "args": ["--transport", "stdio", "--verbose"]
    }
  }
}
```

**Desktop Extension (New):**
- One-click installation
- No JSON configuration needed  
- UI-based settings
- Automatic dependency management