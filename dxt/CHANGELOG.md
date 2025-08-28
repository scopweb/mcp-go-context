# Changelog

All notable changes to the MCP Go Context Desktop Extension will be documented in this file.

## [2.0.0] - 2025-08-28

### Added
- 🎉 **Desktop Extension Support** (.dxt format)
- 🔒 **JWT Authentication** system for HTTP/SSE transports
- 🛡️ **CORS Configurables** - Replace wildcard with secure origin lists
- 🚀 **Streamable HTTP Transport** - MCP 2025-03-26 protocol support
- 📱 **One-click Installation** for Claude Desktop
- ⚙️ **User Configuration** - JWT secrets, config paths, project paths
- 🔐 **Sensitive Configuration** - Secure storage of API keys and secrets

### Enhanced  
- 🧠 **Memory Management** - Improved persistence and search
- 📊 **Project Analysis** - Enhanced AST parsing and metrics
- 🌐 **Documentation Fetching** - Better Context7 integration
- 🛠️ **Tool Registration** - Complete MCP 2025 tool definitions
- 📋 **Protocol Compliance** - Full MCP 2025-03-26 specification

### Security
- ✅ **JWT Token Validation** with expiration and signature verification
- ✅ **CORS Origin Whitelisting** - No more wildcard (`*`) usage
- ✅ **Input Validation** - Enhanced parameter sanitization
- ✅ **Path Traversal Protection** - Secure file operations
- ✅ **Authentication Logs** - Security event monitoring

### Compatibility
- ✅ **Claude Desktop 3.5+** - Full compatibility
- ✅ **Cross-Platform** - Windows, macOS, Linux support
- ✅ **Backward Compatible** - Existing stdio configurations work unchanged
- ✅ **Transport Flexibility** - stdio, HTTP, SSE, Streamable HTTP

### Developer Experience
- 🧪 **Comprehensive Tests** - JWT, CORS, memory, tools testing
- 📚 **Enhanced Documentation** - Security guides and examples
- 🔧 **Development Tools** - Token generation, configuration examples
- 📦 **Easy Distribution** - Single .dxt file installation

## [1.0.0] - Previous Version

### Features
- Basic MCP server functionality
- Project analysis tools
- Memory management
- Simple token authentication
- HTTP/SSE/stdio transports

---

**Installation**: Download `mcp-go-context.dxt` and drag into Claude Desktop  
**Documentation**: See README.md for usage instructions  
**Support**: https://github.com/scopweb/mcp-go-context/issues