# MCP Go Context Server 🚀

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen?style=flat-square)](#)

> **Advanced Context Management for AI Coding Assistants**  
> A high-performance MCP server that provides intelligent project analysis, persistent memory, and hybrid documentation fetching.

## 🔧 Recent Fixes (2025-07-03)

**✅ Claude Desktop Compatibility Fixed**
- **Protocol Issue Resolved**: Fixed JSON-RPC protocol incompatibility with Claude Desktop
- **Auto-Detection**: Intelligent format detection for both JSON-direct and headers-based protocols
- **Stable Connection**: Resolved EOF handling that caused 1-minute disconnections
- **Notification Support**: Added proper handling for `notifications/cancelled` and other client notifications
- **Direct JSON Transport**: Optimized stdio transport for Claude Desktop's expected format

**🎯 What was fixed:**
- ❌ Server disconnecting after ~60 seconds → ✅ Persistent connection
- ❌ "Content-Length is not valid JSON" errors → ✅ Direct JSON protocol
- ❌ Initialize timeouts → ✅ Proper handshake handling
- ❌ EOF terminating process → ✅ Graceful reconnection with retry logic

## ✨ Features

- 🧠 **Persistent Conversation Memory** - Remembers context across sessions
- 📊 **Deep Project Analysis** - AST parsing, dependency mapping, and metrics
- 🌐 **Hybrid Documentation** - Context7 API + local analysis + fallbacks
- ⚡ **High Performance** - Local caching and incremental analysis
- 🔧 **Zero Dependencies** - Single binary, pure Go stdlib
- 🚀 **Multi-Transport** - stdio, HTTP, and SSE support
- ⚙️ **Highly Configurable** - JSON-based configuration system

## 🆚 Why Choose Over Context7?

| Feature | Context7 | MCP Go Context | Advantage |
|---------|----------|----------------|-----------|
| **Offline Analysis** | ❌ | ✅ | Works without internet |
| **Conversation Memory** | ❌ | ✅ | Persistent across sessions |
| **Project Understanding** | ❌ | ✅ | Deep AST analysis |
| **Performance** | API calls | ✅ | Local cache + analysis |
| **Dependencies** | Node.js | ✅ | Single binary |
| **Extensibility** | Limited | ✅ | Modular architecture |
| **Claude Desktop Compatibility** | ✅ | ✅ | **FIXED** - Both work seamlessly |
| **Stability** | Good | ✅ | **IMPROVED** - No disconnections |

## 🔧 Troubleshooting

### Common Issues Fixed

**1. Server Disconnects After 1 Minute**
- ✅ **FIXED** in latest version
- The server now handles EOF gracefully and maintains persistent connections

**2. "Content-Length is not valid JSON" Error**
- ✅ **FIXED** in latest version  
- Implemented direct JSON protocol matching Claude Desktop's expectations

**3. "Request timed out" During Initialize**
- ✅ **FIXED** in latest version
- Added proper JSON-RPC protocol handling with auto-detection

**4. Server Process Exits Unexpectedly**
- ✅ **FIXED** in latest version
- Improved error handling and connection retry logic

### Current Status
- **Stable**: ✅ No more disconnections
- **Compatible**: ✅ Works with Claude Desktop out-of-the-box
- **Persistent**: ✅ Maintains memory across sessions
- **Fast**: ✅ Local analysis without API calls

## 🚀 Quick Start

### Installation

**Recommended - Use Fixed Version:**
```bash
# Download the latest release
go install github.com/scopweb/mcp-go-context@latest

# Or build from source (fixed version)
git clone https://github.com/scopweb/mcp-go-context
cd mcp-go-context
go build -o mcp-context-server.exe cmd/mcp-context-server/main.go
```

### Configuration for Claude Desktop

Add to your `claude_desktop_config.json`:

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

**Note:** The `--verbose` flag is recommended for debugging and provides detailed logging.

### Configuration for Cursor

Add to your `.cursor/mcp.json`:

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

## 🛠️ Available Tools

### 📊 `analyze-project`
Performs comprehensive project analysis with metrics and dependency mapping.

### 🔍 `get-context`
Retrieves intelligent context for your current task with memory integration.

### 📚 `fetch-docs`
Fetches documentation using Context7 API with intelligent fallbacks.

### 💭 `remember-conversation`
Stores important context for future reference with intelligent tagging.

### 🔗 `dependency-analysis`
Analyzes project dependencies with security recommendations.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Made with ❤️ by [ScopWeb](https://scopweb.com)**
