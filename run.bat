@echo off
echo Iniciando MCP Context Server...

cd /d "C:\MCPs\clone\mcp-go-context"

if not exist "bin\mcp-context-server.exe" (
    echo Ejecutable no encontrado. Ejecuta build.bat primero.
    pause
    exit /b 1
)

echo.
echo Selecciona el modo de transporte:
echo 1. STDIO (para Claude Desktop)
echo 2. HTTP (puerto 3000)
echo 3. SSE (puerto 3001)
echo.
set /p choice="Ingresa tu opción (1-3): "

if "%choice%"=="1" (
    echo Iniciando en modo STDIO...
    bin\mcp-context-server.exe --transport stdio
) else if "%choice%"=="2" (
    echo Iniciando en modo HTTP en puerto 3000...
    echo Abre http://localhost:3000/health para verificar
    bin\mcp-context-server.exe --transport http --port 3000
) else if "%choice%"=="3" (
    echo Iniciando en modo SSE en puerto 3001...
    echo Abre http://localhost:3001/sse para verificar
    bin\mcp-context-server.exe --transport sse --port 3001
) else (
    echo Opción inválida
    pause
    exit /b 1
)

pause
