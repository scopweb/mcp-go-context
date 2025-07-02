# MCP Go Context Server 🚀

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen?style=flat-square)](#)

> **Advanced Context Management for AI Coding Assistants**  
> A high-performance MCP server that provides intelligent project analysis, persistent memory, and hybrid documentation fetching.

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

## 🚀 Quick Start

### Installation

```bash
# Download the latest release
go install github.com/scopweb/mcp-go-context@latest

# Or build from source
git clone https://github.com/scopweb/mcp-go-context
cd mcp-go-context
go build -o mcp-context-server.exe
```

### Configuration for Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "mcp-context-server.exe",
      "args": []
    }
  }
}
```

### Configuration for Cursor

Add to your `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "mcp-context-server.exe",
      "args": []
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
