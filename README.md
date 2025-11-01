# MCP Go Context Server 🚀

## 📝 Changelog

### 2025-08-28 - MCP 2025 Security & Desktop Extensions Update 🚀
- 🎉 **Desktop Extensions (.dxt)** - One-click installation for Claude Desktop
- 🔒 **JWT Authentication** - Modern security replacing simple tokens
- 🛡️ **CORS Configurables** - Secure origin whitelisting (no more wildcard `*`)
- 🚀 **Streamable HTTP Transport** - MCP 2025-03-26 protocol compliance
- 📦 **User Configuration** - JWT secrets, config paths with OS keychain
- ⚙️ **Protocol Upgrade** - Full MCP 2025 capabilities and features
- 🧪 **Comprehensive Tests** - JWT, CORS, Streamable transport validation
- 🏗️ **Build Automation** - Cross-platform .dxt package generation
- ✅ **Backward Compatible** - Existing configurations work unchanged

### 2025-08-15  
- Refuerzo de seguridad en validación de parámetros y rutas en todos los módulos principales.
- Mejoras de robustez y control de errores en memory, server y tools.
- Tests unitarios automáticos para todas las funciones públicas de memoria y tools.
- Validación estricta de nombres, rutas y argumentos en handlers y API.
- LRU y control de sesiones en memoria.
- Restricción de lectura de archivos y límites de tamaño en documentación local.
- Autenticación opcional por token para transportes HTTP/SSE.
- Todos los tests en `test/` pasan correctamente.
- Pruebas de integración para autenticación HTTP/SSE (JSON-RPC) añadidas.
- Nuevas tools expuestas: `memory-get`, `memory-search`, `memory-recent`, `memory-clear`, `config-get-project-paths`.
- Documentación de autenticación por token con `MCP_SERVER_TOKEN`.


[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-2025--03--26-blue?style=flat-square)](https://modelcontextprotocol.io/)
[![DXT](https://img.shields.io/badge/Desktop%20Extensions-✓-green?style=flat-square)](#)
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
- 🚀 **Multi-Transport** - stdio, HTTP, SSE, and Streamable HTTP support
- ⚙️ **Highly Configurable** - JSON-based configuration system
- 🎯 **Desktop Extensions** - One-click installation via .dxt files
- 🔒 **Enterprise Security** - JWT authentication and CORS protection
- 📱 **MCP 2025 Ready** - Full protocol compliance with latest features

## 🚀 Installation Methods

### Method 1: Desktop Extension (Recommended) 
```bash
1. Download mcp-go-context.dxt
2. Drag & drop into Claude Desktop
3. Configure optional settings via UI
4. Start using tools immediately!
```

### Method 2: Traditional Installation
```bash
# Download release or build from source
go install github.com/scopweb/mcp-go-context@latest

# Or build from source
git clone https://github.com/scopweb/mcp-go-context
cd mcp-go-context
go build -o mcp-context-server.exe cmd/mcp-context-server/main.go
```

## ⚙️ Configuration

### For Claude Desktop (stdio - no changes needed)
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

### For Advanced HTTP/SSE Usage
```json
{
  "transport": {
    "type": "streamable-http",
    "port": 3000
  },
  "security": {
    "auth": {
      "enabled": true,
      "method": "jwt",
      "secret": "${MCP_JWT_SECRET}",
      "expiry": "1h"
    },
    "cors": {
      "enabled": true,
      "origins": ["app://claude-desktop", "https://localhost:3000"]
    }
  }
}
```

## 🔐 Security Features

### JWT Authentication (HTTP/SSE only)
- **Environment Variable**: `MCP_JWT_SECRET`
- **Token Generation**: Use `auth-generate-token` tool
- **Header Format**: `Authorization: Bearer <jwt-token>`
- **Automatic Expiration**: Configurable (default: 1 hour)

### CORS Protection
- **Whitelist Origins**: No more wildcard `*` usage
- **Claude Desktop**: `app://claude-desktop` automatically allowed
- **Wildcard Patterns**: Support for `*.yourdomain.com`
- **Preflight Handling**: Full OPTIONS request support

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

---

## 🎯 **Honest Truth: When You Should (and Shouldn't) Use This MCP**

### ✅ **USE THIS MCP IF:**

1. **You work in distributed teams** (3+ developers)
   - Need shared project context across team members
   - Want centralized memory management
   - HTTP API for CI/CD automation

2. **You have large monorepos** (50+ services/modules)
   - Need AST-based analysis for quick navigation
   - Dependency mapping is critical
   - Performance optimization matters

3. **You need automation & scripting**
   - Using Claude in CI/CD pipelines
   - Programmatic access via HTTP/SSE
   - Enterprise security requirements (JWT, CORS)

4. **You want audit logging & compliance**
   - Need security audit trails
   - OWASP/CWE compliance required
   - Enterprise deployments

5. **You're building tools/frameworks**
   - Need a context provider for other tools
   - Want to extend MCP capabilities
   - Research on MCPs and AI context management

---

### ❌ **DON'T USE THIS MCP IF:**

1. **You're a solo developer** on small/medium projects
   - **Better:** Use Claude Desktop's native `CLAUDE.md`
   - Claude already remembers context automatically
   - No need for extra complexity

2. **You just want to remember things**
   - **Better:** Use Claude's built-in memory or `CLAUDE.md`
   - Simpler to use and manage
   - Less overhead

3. **You think it magically solves context problems**
   - It's NOT AI-powered context ranking
   - It's regex-based search + heuristics
   - Claude Desktop is already quite smart with context

4. **You need semantic code understanding**
   - This MCP only does basic AST parsing
   - No ML-based code analysis
   - Not a replacement for actual code intelligence

---

### 📊 **Real-World Decision Matrix**

| Your Situation | Solution | Why |
|---|---|---|
| Solo dev + small project | ❌ Skip MCP | Use `CLAUDE.md` natively |
| Solo dev + large project | 🟡 Maybe | Only if monorepo needs mapping |
| Team <5 devs | 🟡 Maybe | Workspace sharing might be enough |
| Team 5+ devs | ✅ Use MCP | Shared context is valuable |
| CI/CD automation | ✅ Use MCP | HTTP API is essential |
| Enterprise/Compliance | ✅ Use MCP | Security features needed |
| Just learning/exploring | ✅ Use MCP | Great for learning MCPs |

---

### 💬 **What Users Actually Say:**

> "**I don't see benefits after a day of use"**
> ✅ **Solution:** That's normal. Read the [FAQ](./docs/faq/FAQ-01-no-benefits.md). This MCP is for specific use cases, not everyone.

> "**Why is my memory folder empty?"**
> ✅ **Solution:** You need to actively use `remember-conversation` tool. It's not automatic. Read [FAQ #2](./docs/faq/FAQ-02-empty-memory.md).

> "**How do I use this correctly?"**
> ✅ **Solution:** Check [FAQ #3](./docs/faq/FAQ-03-correct-usage.md) for practical workflows.

---

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

## 🛠️ Available Tools (11 Total)

### 📊 Analysis Tools
- **`analyze-project`** - Comprehensive project analysis with metrics and dependency mapping
- **`dependency-analysis`** - Project dependency analysis with security recommendations
- **`config-get-project-paths`** - Get configured project paths

### 🔍 Context & Documentation
- **`get-context`** - Intelligent context retrieval with memory integration
- **`fetch-docs`** - Documentation fetching using Context7 API with intelligent fallbacks

### 💭 Memory Management
- **`remember-conversation`** - Store important context for future reference
- **`memory-get`** - Retrieve a memory item by key
- **`memory-search`** - Search memories by text and/or tags with result limits
- **`memory-recent`** - Get recent memories (configurable limit)
- **`memory-clear`** - Clear all memories (requires explicit confirmation)

### 🔐 Security & Development
- **`auth-generate-token`** - Generate JWT tokens for development/testing

## 🚀 Quick Start

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

### 🧠 `memory-get`
Obtiene un ítem de memoria por clave.

Input mínimo:
```
{"key":"mi-clave"}
```

### 🔎 `memory-search`
Busca memorias por texto y/o etiquetas, con límite de resultados.

Input opcional:
```
{"query":"texto","tags":["t1","t2"],"limit":10}
```

### 🕒 `memory-recent`
Devuelve memorias recientes (hasta `limit`).

Input opcional:
```
{"limit":10}
```

### ⚠️ `memory-clear`
Elimina todas las memorias. Requiere confirmación explícita.

Input requerido:
```
{"confirm":"YES_I_UNDERSTAND"}
```

### 🗂️ `config-get-project-paths`
Lista las rutas de proyecto configuradas actualmente.
```
{}
```

## 🚀 Transport Options

### 📡 Available Transports
- **`stdio`** - Standard I/O (Claude Desktop default, no auth required)
- **`http`** - HTTP JSON-RPC (with JWT auth support)
- **`sse`** - Server-Sent Events (real-time streaming)  
- **`streamable-http`** - Hybrid HTTP + SSE (MCP 2025-03-26 protocol)

### 🎯 Transport Usage
```bash
# Claude Desktop (recommended)
./mcp-context-server --transport stdio --verbose

# HTTP with authentication  
./mcp-context-server --transport http --port 3000

# Streamable HTTP (MCP 2025)
./mcp-context-server --transport streamable-http --port 3000
```

### 🔧 Protocol Support
- **MCP Version**: `2025-03-26` (latest)
- **JSON-RPC**: `2.0` compliant
- **Capabilities**: Tools, sampling, roots, resources
- **Security**: JWT authentication, CORS protection

## 🧪 Tests

- **Full Test Suite**: `go test ./...`
- **JWT Authentication**: `go test ./test -run TestJWTAuthSimple`
- **CORS Security**: `go test ./test -run TestCORS` 
- **Streamable Transport**: `go test ./test -run TestStreamable`
- **Integration Tests**: HTTP/SSE with authentication
- **Memory & Tools**: Unit tests for all public functions

## 📚 Documentation & Support

### 📖 **[Complete User Manual](./docs/MANUAL.md)**
Step-by-step guide with practical examples and workflows.

### ❓ **[FAQ Collection](./docs/faq/README.md)**
Answers to the most common questions and problems:

- **[FAQ #1](./docs/faq/FAQ-01-no-benefits.md)** - "No veo beneficios del MCP después de un día de uso"
- **[FAQ #2](./docs/faq/FAQ-02-empty-memory.md)** - "¿Por qué está vacía mi carpeta .mcp-context/memory?"
- **[FAQ #3](./docs/faq/FAQ-03-correct-usage.md)** - "¿Cómo usar las herramientas del MCP correctamente?"
- **[FAQ #4](./docs/faq/FAQ-04-disconnection.md)** - "El MCP se desconecta después de 60 segundos"
- **[FAQ #5](./docs/faq/FAQ-05-claude-config.md)** - "¿Cómo configurar correctamente el MCP en Claude Desktop?"
- **[FAQ #6](./docs/faq/FAQ-06-use-cases.md)** - "¿Cuáles son los casos de uso prácticos del MCP?"
- **[FAQ #7](./docs/faq/FAQ-07-troubleshooting.md)** - "Troubleshooting: Problemas comunes y soluciones"

### 🆘 **Need Help?**
1. Check the [FAQ collection](./docs/faq/README.md) first
2. Search [existing issues](https://github.com/scopweb/mcp-go-context/issues?q=label%3Afaq)
3. Open a [new issue](https://github.com/scopweb/mcp-go-context/issues/new) if your problem isn't covered

## 🤝 Contributors

This project represents a unique collaboration between human creativity and AI assistance:

### 👨‍💻 [ScopWeb](https://scopweb.com) - Project Lead
- Original project conception and v1.0.0 development
- Infrastructure, deployment, and community management
- Project direction and strategic decisions

### 🤖 Claude (Anthropic AI Assistant) - AI Development Partner
- v2.0.0 major architecture redesign and implementation
- JWT authentication system and security enhancements
- DXT format specification and comprehensive documentation
- Multi-language support and test suite development

**See [CONTRIBUTORS.md](./docs/CONTRIBUTORS.md) for detailed contribution information.**

## 📚 Additional Documentation

- **[Performance Optimizations](./docs/OPTIMIZATIONS.md)** - Applied optimizations and benchmarks
- **[JWT Security Guide](./docs/JWT-SECURITY-GUIDE.md)** - JWT authentication setup and best practices
- **[CORS Configuration](./docs/CORS-SECURITY-GUIDE.md)** - CORS security configuration guide
- **[MCP 2025 Upgrade Guide](./docs/MCP-2025-UPGRADE-GUIDE.md)** - Protocol upgrade information
- **[Contributing Guidelines](./docs/CONTRIBUTING.md)** - How to contribute to the project
- **[Project Report](./docs/PROJECT_REPORT.md)** - Detailed project analysis and architecture

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Made with ❤️ by [ScopWeb](https://scopweb.com) in collaboration with Claude AI**
