@echo off
echo.
echo 🚀 COMPILACION FINAL - MCP Context Server
echo ==========================================

echo.
echo 📋 Verificando estructura del proyecto...
dir /b internal 2>nul >nul && echo ✅ Directorio internal encontrado || echo ❌ Directorio internal no encontrado
dir /b cmd 2>nul >nul && echo ✅ Directorio cmd encontrado || echo ❌ Directorio cmd no encontrado
if exist go.mod echo ✅ go.mod encontrado || echo ❌ go.mod no encontrado

echo.
echo 🔧 Verificando dependencias...
go mod tidy
if %ERRORLEVEL% neq 0 (
    echo ❌ Error en go mod tidy
    exit /b 1
)
echo ✅ Dependencias verificadas

echo.
echo 🔨 Compilando servidor principal...
if not exist bin mkdir bin
go build -ldflags "-X main.version=1.0.0 -X main.commit=final" -o bin/mcp-context-server.exe ./cmd/mcp-context-server

if %ERRORLEVEL% equ 0 (
    echo ✅ Compilación del servidor exitosa
    echo 📍 Binary: bin/mcp-context-server.exe
) else (
    echo ❌ Error en compilación del servidor
    exit /b 1
)

echo.
echo 🔨 Compilando pruebas...
go build -o test-complete.exe test-complete.go
if %ERRORLEVEL% equ 0 (
    echo ✅ Compilación de pruebas exitosa
) else (
    echo ❌ Error en compilación de pruebas
    exit /b 1
)

echo.
echo 🧪 Ejecutando tests comprehensivos...
echo ==========================================
test-complete.exe

echo.
echo ==========================================
echo.
echo 🎯 COMPILACION COMPLETADA
echo.
echo 📦 Archivos generados:
if exist bin\mcp-context-server.exe (
    echo   ✅ bin\mcp-context-server.exe
    for %%A in (bin\mcp-context-server.exe) do echo      Tamaño: %%~zA bytes
)
if exist test-complete.exe (
    echo   ✅ test-complete.exe  
    for %%A in (test-complete.exe) do echo      Tamaño: %%~zA bytes
)

echo.
echo 🚀 El MCP Context Server está listo para usar!
echo.
echo 📋 Próximos pasos:
echo   1. Configurar en Claude Desktop:
echo      {"mcpServers": {"mcp-context": {"command": "C:\\ruta\\bin\\mcp-context-server.exe"}}}
echo   2. Reiniciar Claude Desktop
echo   3. Usar los tools: analyze-project, get-context, fetch-docs, remember-conversation, dependency-analysis
echo.
echo 📖 Ver README.md para instrucciones detalladas
echo.

pause
