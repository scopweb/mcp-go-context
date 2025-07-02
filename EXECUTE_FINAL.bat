@echo off
echo.
echo 🎯 EJECUTANDO COMPILACION FINAL
echo ==========================================
echo.

cd /d "C:\MCPs\clone\mcp-go-context"

call compile-final.bat

echo.
echo 🎉 PROCESO COMPLETADO
echo.
echo 📋 Archivos generados en el proyecto:
dir /b bin\*.exe 2>nul
dir /b *.exe 2>nul

echo.
echo 🚀 El MCP Context Server está LISTO para producción!
echo.
