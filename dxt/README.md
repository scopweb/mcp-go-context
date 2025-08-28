# MCP Go Context - Desktop Extension

This Desktop Extension (.dxt) provides one-click installation of the MCP Go Context server for Claude Desktop.

## Features

- 🧠 **Persistent Conversation Memory** - Remembers context across sessions
- 📊 **Deep Project Analysis** - AST parsing, dependency mapping, and metrics
- 🌐 **Hybrid Documentation** - Context7 API + local analysis + fallbacks
- ⚡ **High Performance** - Local caching and incremental analysis
- 🔧 **Zero Dependencies** - Single binary, pure Go stdlib
- 🚀 **Multi-Transport** - stdio, HTTP, SSE, and Streamable HTTP support
- ⚙️ **Security Enhanced** - JWT authentication and CORS protection

## Installation

1. Download the `mcp-go-context.dxt` file
2. Double-click or drag into Claude Desktop
3. Configure optional settings (JWT secret, config path)
4. Start using the tools immediately!

## Available Tools

### Analysis Tools
- `analyze-project` - Comprehensive project analysis
- `dependency-analysis` - Dependency analysis with security recommendations
- `config-get-project-paths` - Get configured project paths

### Context & Documentation
- `get-context` - Intelligent context retrieval
- `fetch-docs` - Documentation fetching with fallbacks

### Memory Management
- `remember-conversation` - Store important context
- `memory-get` - Retrieve memory by key
- `memory-search` - Search memories by query/tags
- `memory-recent` - Get recent memories
- `memory-clear` - Clear all memories (with confirmation)

### Security & Development
- `auth-generate-token` - Generate JWT tokens for development

## Configuration

The extension supports optional configuration:

- **JWT Secret**: For HTTP/SSE authentication (leave empty for stdio-only)
- **Config Path**: Path to custom configuration file
- **Project Paths**: Directories to analyze (defaults to current directory)

## Compatibility

- **Windows**: ✅ Supported
- **macOS**: ✅ Supported  
- **Linux**: ✅ Supported
- **Claude Desktop**: 3.5+
- **Go Runtime**: Not required (bundled executable)

## Protocol Support

- **MCP Protocol**: 2025-03-26
- **Transport**: stdio (default), HTTP, SSE, Streamable HTTP
- **Security**: JWT authentication, CORS protection
- **Features**: Project analysis, memory persistence, documentation fetching

## License

MIT License - see LICENSE file for details.

## Author

ScopWeb - https://scopweb.com