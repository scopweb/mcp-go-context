#!/bin/bash
echo "🏗️ Building MCP Go Context Desktop Extension (.dxt)"

cd "$(dirname "$0")"

echo "📦 Creating dxt build directory..."
rm -rf dxt-build
mkdir -p dxt-build/bin

echo "🔨 Building Go binary..."
go build -ldflags "-X main.version=2.0.0 -X main.commit=$(date +%Y%m%d)" -o dxt-build/bin/mcp-context-server cmd/mcp-context-server/main.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "📋 Copying extension files..."
cp dxt/manifest.json dxt-build/
cp dxt/README.md dxt-build/
cp dxt/package.json dxt-build/
cp dxt/CHANGELOG.md dxt-build/
cp LICENSE dxt-build/

echo "🎨 Creating placeholder assets..."
mkdir -p dxt-build/screenshots
touch dxt-build/icon.png
touch dxt-build/screenshots/project-analysis.png
touch dxt-build/screenshots/context-retrieval.png
touch dxt-build/screenshots/memory-management.png

echo "📁 DXT Package Structure:"
ls -la dxt-build/

echo "🎯 Creating .dxt archive..."
cd dxt-build
zip -r ../mcp-go-context.dxt ./*
cd ..

if [ -f "mcp-go-context.dxt" ]; then
    echo "✅ Desktop Extension created: mcp-go-context.dxt"
    echo "📏 File size: $(ls -lh mcp-go-context.dxt | awk '{print $5}')"
    echo ""
    echo "🚀 Installation:"
    echo "  1. Drag mcp-go-context.dxt into Claude Desktop"
    echo "  2. Configure optional settings"
    echo "  3. Start using MCP tools!"
    echo ""
else
    echo "❌ Failed to create .dxt file!"
    exit 1
fi

echo "🧹 Cleaning up build directory..."
rm -rf dxt-build

echo "✨ Build complete!"