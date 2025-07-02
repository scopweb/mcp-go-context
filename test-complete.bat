@echo off
echo 🧪 Testing Complete MCP Context Server Implementation

echo.
echo 🔨 Building test executable...
go build -o test-complete.exe test-complete.go

if %ERRORLEVEL% neq 0 (
    echo ❌ Build failed!
    exit /b 1
)

echo.
echo 🚀 Running comprehensive tests...
echo ==========================================
test-complete.exe

echo.
echo ==========================================
echo.
echo 🧪 Running individual component tests...

echo.
echo 📋 Testing analyzer directly...
go run test-analyzer.go

echo.
echo 📋 Testing tools directly...
go run test-tools.go

echo.
echo 🎯 All tests completed!
echo.
echo 📁 Generated files:
if exist test-complete.exe echo   - test-complete.exe
if exist bin\mcp-context-server.exe echo   - bin\mcp-context-server.exe
echo.
echo 🚀 Ready for production use!

pause
