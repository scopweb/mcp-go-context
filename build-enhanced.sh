#!/bin/bash

echo "🔨 Building MCP Context Server..."

# Set build variables
VERSION="1.0.0"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}"

# Create bin directory
mkdir -p bin

# Build for current platform
echo "📦 Building for current platform..."
go build -ldflags "${LDFLAGS}" -o bin/mcp-context-server ./cmd/mcp-context-server

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "📍 Binary: bin/mcp-context-server"
    echo "🔖 Version: ${VERSION}"
    echo "🔧 Commit: ${COMMIT}"
    
    # Test basic functionality
    echo ""
    echo "🧪 Testing basic functionality..."
    ./bin/mcp-context-server --version
else
    echo "❌ Build failed!"
    exit 1
fi

echo ""
echo "🚀 Ready to run! Try:"
echo "  ./bin/mcp-context-server --help"
echo "  ./bin/mcp-context-server --transport stdio"
