@echo off
echo 🚀 MCP Go Context Server - Build Script
echo ========================================

cd /d "C:\MCPs\clone\mcp-go-context"

echo 📦 Running go mod tidy...
go mod tidy
if %errorlevel% neq 0 (
    echo ❌ go mod tidy failed
    pause
    exit /b 1
)

echo 🧪 Running final test...
go run test-final.go
if %errorlevel% neq 0 (
    echo ❌ Final test failed
    pause
    exit /b 1
)

echo 🏗️ Building production binary...
go build -ldflags="-s -w" -o mcp-context-server.exe main.go
if %errorlevel% neq 0 (
    echo ❌ Build failed
    pause
    exit /b 1
)

echo ✅ Build successful!
echo 📁 Binary location: %CD%\mcp-context-server.exe
echo 📊 File size:
dir mcp-context-server.exe | findstr mcp-context-server.exe

echo.
echo 🎉 MCP Go Context Server is ready for use!
echo.
echo 📋 Next steps:
echo 1. Copy mcp-context-server.exe to your desired location
echo 2. Add to your MCP client configuration
echo 3. Start using the advanced context features!
echo.

pause
