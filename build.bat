@echo off
echo 🔨 Building MCP Context Server...

cd /d C:\MCPs\clone\mcp-go-context

echo 📦 Running go mod tidy...
go mod tidy

echo 🏗️ Building main server...
go build -o bin\mcp-context-server.exe cmd\mcp-context-server\main.go

if %errorlevel% == 0 (
    echo ✅ Build successful!
    echo 📄 Binary created at: bin\mcp-context-server.exe
    echo.
    echo 🚀 Usage:
    echo   bin\mcp-context-server.exe --transport=stdio
    echo   bin\mcp-context-server.exe --transport=http --port=3000
    echo.
) else (
    echo ❌ Build failed!
    exit /b 1
)

pause
