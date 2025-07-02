# Contributing to MCP Context Server

First off, thank you for considering contributing to MCP Context Server! 🎉

## 🚀 Quick Start

### Development Setup
1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/mcp-context-server.git
   cd mcp-context-server
   ```
3. **Install dependencies**:
   ```bash
   go mod download
   ```
4. **Run tests** to ensure everything works:
   ```bash
   go run test-complete.go
   ```

## 🛠️ Development Workflow

### Making Changes
1. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** with proper testing

3. **Run the test suite**:
   ```bash
   # Comprehensive tests
   go run test-complete.go
   
   # Unit tests
   go test ./...
   
   # Build verification
   go build ./cmd/mcp-context-server
   ```

4. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add amazing new feature"
   ```

5. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

6. **Create a Pull Request** on GitHub

### Commit Message Convention
We follow conventional commits:
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation updates
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance tasks

## 🧪 Testing Guidelines

### Test Requirements
- All new features must include tests
- Bug fixes should include regression tests
- Test coverage should be maintained above 90%

### Running Tests
```bash
# All tests
go test ./...

# Specific package
go test ./internal/analyzer

# With coverage
go test -cover ./...

# Integration tests
go run test-complete.go
```

## 📝 Code Style

### Go Guidelines
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions small and focused

### Project Structure
```
internal/
├── analyzer/     # Project analysis logic
├── config/       # Configuration management
├── memory/       # Memory and persistence
├── server/       # MCP server implementation
├── tools/        # Tool handlers
└── transport/    # Transport layer
```

## 🎯 Areas for Contribution

### High Priority
- [ ] **Language Support**: Add analyzers for new programming languages
- [ ] **Performance**: Optimize file analysis and memory usage
- [ ] **Documentation**: Improve tool descriptions and examples
- [ ] **Testing**: Expand test coverage and add edge cases

### Medium Priority
- [ ] **Features**: New tool implementations
- [ ] **Integration**: Support for additional dependency managers
- [ ] **UI/UX**: Better error messages and user experience
- [ ] **Platform**: Cross-platform compatibility improvements

### Low Priority
- [ ] **Optimization**: Code refactoring and cleanup
- [ ] **Documentation**: Additional examples and tutorials
- [ ] **CI/CD**: Automated testing and deployment
- [ ] **Monitoring**: Performance metrics and logging

## 🐛 Reporting Issues

### Bug Reports
When reporting bugs, please include:
- **OS and version** (Windows 11, macOS 14, Ubuntu 22.04, etc.)
- **Go version** (`go version`)
- **MCP Context Server version**
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Log output** if available
- **Minimal reproduction case** if possible

### Feature Requests
For feature requests, please provide:
- **Clear description** of the feature
- **Use case** and motivation
- **Proposed implementation** (if you have ideas)
- **Alternative solutions** considered

## 🚀 Adding New Tools

### Tool Implementation Guide
1. **Create handler function** in `internal/tools/`:
   ```go
   func MyNewToolHandler(args json.RawMessage, server interface{}) (interface{}, error) {
       // Implementation
   }
   ```

2. **Register the tool** in `internal/server/server.go`:
   ```go
   s.tools.Register(&tools.Tool{
       Name:        "my-new-tool",
       Description: "Description of what it does",
       InputSchema: schema,
       Handler:     tools.MyNewToolHandler,
   })
   ```

3. **Add tests** in test files
4. **Update documentation**

### Tool Best Practices
- **Validate inputs** thoroughly
- **Handle errors** gracefully
- **Return structured responses**
- **Include helpful error messages**
- **Add comprehensive tests**

## 🌍 Adding Language Support

### Analyzer Extension
1. **Add language detection** in `detectLanguage()` function
2. **Implement parser** for the language:
   ```go
   func (a *ProjectAnalyzer) analyzeNewLanguageFile(path string, info *FileInfo) error {
       // Language-specific parsing logic
   }
   ```
3. **Add dependency support** if applicable
4. **Include test cases** with sample files
5. **Update documentation**

### Supported Elements
When adding language support, try to extract:
- **Imports/Dependencies**: External modules used
- **Functions/Methods**: Function definitions
- **Types/Classes**: Type definitions
- **Variables/Constants**: Global declarations
- **Exports**: Public API elements

## 📚 Documentation

### Documentation Standards
- **Clear and concise** explanations
- **Code examples** for all features
- **Up-to-date** information
- **Proper formatting** with Markdown

### Documentation Types
- **README.md**: Overview and quick start
- **API Documentation**: Tool specifications
- **Developer Guide**: Architecture and internals
- **User Guide**: Usage examples and tutorials

## 🔄 Release Process

### Version Numbering
We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes

### Release Checklist
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Version bumped in relevant files
- [ ] Changelog updated
- [ ] Binary builds tested on all platforms

## 💬 Getting Help

### Community Channels
- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and community chat
- **Code Reviews**: We provide detailed feedback on PRs

### Mentorship
New contributors are welcome! We're happy to:
- Review your code and provide feedback
- Help with Go best practices
- Guide you through the contribution process
- Pair program on complex features

## 🏆 Recognition

### Contributors
All contributors are recognized in:
- GitHub contributor list
- Project documentation
- Release notes for significant contributions

### Contribution Types
We value all types of contributions:
- **Code**: New features and bug fixes
- **Documentation**: Improving clarity and examples
- **Testing**: Expanding test coverage
- **Issues**: Reporting bugs and suggesting features
- **Reviews**: Helping review pull requests
- **Community**: Helping other users

## 📋 Code of Conduct

### Our Standards
- **Be respectful** and inclusive
- **Be helpful** and supportive
- **Be constructive** in feedback
- **Focus on the code**, not the person

### Enforcement
- Issues will be addressed promptly
- Violations may result in temporary or permanent bans
- Contact maintainers for serious concerns

## 🎉 Thank You!

Your contributions make MCP Context Server better for everyone. Whether you're fixing a typo, adding a feature, or helping other users, every contribution matters!

Happy coding! 🚀
