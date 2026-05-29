@echo off
setlocal enabledelayedexpansion

echo ========================================
echo   MCP Context Server - Build Script
echo ========================================
echo.

:: Check if Go is installed
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go is not installed or not in PATH.
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

:: Get version info
set VERSION=dev
for /f "tokens=*" %%i in ('git describe --tags --always 2^>nul') do set VERSION=%%i
if "%VERSION%"=="dev" (
    for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set VERSION=dev-%%i
)

echo [INFO] Version: %VERSION%
echo [INFO] Building for Windows...

:: Create bin directory if it doesn't exist
if not exist bin mkdir bin

:: Build the server (dashboard is fully embedded inside the .exe)
echo [INFO] Compiling main binary + embedded dashboard...
echo [INFO] El .exe resultante será: bin\mcp-context-server.exe (el que utilizas)
go build -ldflags "-X github.com/scopweb/mcp-go-context/internal/buildinfo.Version=%VERSION%" -o bin/mcp-context-server.exe ./cmd/mcp-context-server

if %ERRORLEVEL% neq 0 (
    echo.
    echo [ERROR] Build failed!
    pause
    exit /b 1
)

echo.
echo [SUCCESS] Build completed successfully!
echo.
echo   ================================================
echo     .EXE LISTO PARA USAR:
echo     bin\mcp-context-server.exe
echo   ================================================
echo.
echo   Este es el archivo que utilizas en Claude Desktop / tu cliente MCP.
echo.
echo   Version: %VERSION%
echo.
echo   Nota: El dashboard está embebido dentro del .exe.
echo         No necesitas compilar nada adicional.
echo.
echo   Ejemplos de uso:
echo     bin\mcp-context-server.exe --transport http     (para acceder al dashboard)
echo     bin\mcp-context-server.exe --transport stdio    (para Claude Desktop)
echo.
pause
