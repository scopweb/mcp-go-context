@echo off
echo 🔨 Building MCP Context Server...

rem Set build variables
set VERSION=1.0.0
for /f %%i in ('git rev-parse --short HEAD 2^>nul') do set COMMIT=%%i
if "%COMMIT%"=="" set COMMIT=dev

rem Build flags
set LDFLAGS=-X main.version=%VERSION% -X main.commit=%COMMIT%

rem Create bin directory
if not exist bin mkdir bin

rem Build for current platform
echo 📦 Building for current platform...
go build -ldflags "%LDFLAGS%" -o bin/mcp-context-server.exe ./cmd/mcp-context-server

if %ERRORLEVEL% equ 0 (
    echo ✅ Build successful!
    echo 📍 Binary: bin/mcp-context-server.exe
    echo 🔖 Version: %VERSION%
    echo 🔧 Commit: %COMMIT%
    
    rem Test basic functionality
    echo.
    echo 🧪 Testing basic functionality...
    bin\mcp-context-server.exe --version
    
    echo.
    echo 🚀 Ready to run! Try:
    echo   bin\mcp-context-server.exe --help
    echo   bin\mcp-context-server.exe --transport stdio
) else (
    echo ❌ Build failed!
    exit /b 1
)
