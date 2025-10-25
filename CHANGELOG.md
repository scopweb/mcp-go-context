# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.1] - 2025-10-25

### Performance Optimizations
- **Go Version**: Updated from 1.21 to 1.23 for latest compiler optimizations
- **Regex Pre-compilation**: 13 regular expressions now compiled at startup
  - 2-5ms faster per regex-heavy function call
  - Affects `analyzeQuery()`, `generateTags()`, and validation functions
- **String Formatting**: Optimized 22+ `fmt.Sprintf` calls to `fmt.Fprintf`
  - 20-30% fewer memory allocations in response generation
  - Reduced garbage collection pressure by 15-20%
- **I/O Safety**: Added `io.LimitReader` to prevent hangs on large files
  - Documentation search: 1MB limit
  - HTTP API responses: 5MB limit
  - SSE requests: 10MB limit

### Project Organization
- **Documentation**: Moved 10 documentation files to `/docs/` directory
  - ANALISIS_FUNCIONAL.md
  - CLAUDE_SETUP.md
  - CONTRIBUTING.md
  - CONTRIBUTORS.md
  - CORS-SECURITY-GUIDE.md
  - JWT-SECURITY-GUIDE.md
  - MANUAL.md
  - MCP-2025-UPGRADE-GUIDE.md
  - OPTIMIZATIONS.md (new)
  - PROJECT_REPORT.md
- **Root Cleanup**: Removed temporary files and old builds
  - Deleted: `nul`, `mcp-context-server.exe` (duplicate)
  - Deleted: `mcp-go-context.dxt` (old build)
  - Removed: `dxt-build/` temporary directory
- **Git Configuration**: Updated and simplified `.gitignore`
- **README**: Updated with new documentation structure and links

### Testing & Verification
- ✅ All 22 tests passing
- ✅ Binary size: 11MB
- ✅ Benchmark: Memory search <1ms for 5,000 items
- ✅ Cross-platform build verified

### Documentation
- Created `CHANGELOG.md` for version tracking
- Created `docs/OPTIMIZATIONS.md` with detailed performance analysis
- Updated `README.md` badges and documentation links

## [2.0.0] - 2025-08-28

### MCP 2025 Security & Desktop Extensions Update 🚀

#### New Features
- 🎉 **Desktop Extensions (.dxt)** - One-click installation for Claude Desktop
  - Drag & drop installation
  - UI-based configuration
  - Automatic dependency management
  - Cross-platform build scripts (`build-dxt.sh`, `build-dxt.bat`)

#### Security Enhancements
- 🔒 **JWT Authentication** - Modern security replacing simple tokens
  - Environment variable: `MCP_JWT_SECRET`
  - Token generation tool: `auth-generate-token`
  - Automatic expiration (configurable, default: 1 hour)
  - Header format: `Authorization: Bearer <token>`
- 🛡️ **CORS Configuration** - Secure origin whitelisting
  - No more wildcard `*` usage
  - Claude Desktop: `app://claude-desktop` auto-allowed
  - Wildcard pattern support: `*.yourdomain.com`
  - Full OPTIONS preflight handling

#### Protocol & Transport
- 🚀 **Streamable HTTP Transport** - MCP 2025-03-26 protocol compliance
  - Hybrid HTTP + SSE transport
  - Full bidirectional communication
  - Enhanced capabilities support
- ⚙️ **Protocol Upgrade** - Full MCP 2025 capabilities
  - Latest JSON-RPC 2.0 implementation
  - Tools, sampling, roots, resources
  - Enhanced error handling

#### Testing & Quality
- 🧪 **Comprehensive Tests**
  - JWT authentication tests
  - CORS security validation
  - Streamable transport tests
  - Integration tests for HTTP/SSE
  - Memory and tools unit tests
- 🏗️ **Build Automation** - Cross-platform .dxt package generation

#### Compatibility
- ✅ **Backward Compatible** - Existing configurations work unchanged
- 📦 **User Configuration** - JWT secrets, config paths with OS keychain support

## [1.5.0] - 2025-08-15

### Security Enhancements
- Reinforced parameter and path validation across all main modules
- Strict validation of names, paths, and arguments in handlers and API
- File read restrictions and size limits for local documentation
- Optional token authentication for HTTP/SSE transports

### Reliability Improvements
- Enhanced robustness and error handling in memory, server, and tools
- LRU and session control in memory management
- All tests in `test/` passing correctly
- Integration tests for HTTP/SSE authentication (JSON-RPC)

### New Tools
- `memory-get` - Retrieve memory item by key
- `memory-search` - Search memories by text and/or tags
- `memory-recent` - Get recent memories
- `memory-clear` - Clear all memories (with confirmation)
- `config-get-project-paths` - List configured project paths

### Documentation
- Token authentication documentation with `MCP_SERVER_TOKEN`
- Automated unit tests for all public memory and tools functions

## [1.0.0] - 2025-07-03

### Initial Release

#### Core Features
- 🧠 **Persistent Conversation Memory** - Context across sessions
- 📊 **Deep Project Analysis** - AST parsing, dependency mapping
- 🌐 **Hybrid Documentation** - Context7 API + local analysis
- ⚡ **High Performance** - Local caching and incremental analysis
- 🔧 **Zero Dependencies** - Single binary, pure Go stdlib

#### Transport Support
- **stdio** - Standard I/O for Claude Desktop
- **HTTP** - HTTP JSON-RPC
- **SSE** - Server-Sent Events for streaming

#### Tools
- `analyze-project` - Comprehensive project analysis
- `get-context` - Intelligent context retrieval
- `fetch-docs` - Documentation fetching
- `remember-conversation` - Context storage
- `dependency-analysis` - Dependency analysis

#### Bug Fixes
- ✅ Fixed Claude Desktop compatibility
  - Protocol issue resolved
  - Auto-detection for JSON-direct and headers-based protocols
  - Stable connection (no 1-minute disconnections)
  - Proper notification support
  - Direct JSON transport optimization

---

## Version History

- **2.0.1** (2025-10-25) - Performance optimizations and project organization
- **2.0.0** (2025-08-28) - MCP 2025, JWT auth, CORS, Desktop Extensions
- **1.5.0** (2025-08-15) - Security enhancements and new memory tools
- **1.0.0** (2025-07-03) - Initial release with Claude Desktop compatibility fixes
