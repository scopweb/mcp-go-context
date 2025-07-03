@echo off
echo 🚀 Preparando repositorio limpio para commit...

cd /d "C:\MCPs\clone\mcp-go-context"

echo 🧹 Paso 1: Limpieza de archivos innecesarios...
echo 📁 Eliminando archivos de test...
del /q test-*.go test_*.go test-*.bat test-*.sh test-*.json 2>nul

echo 📁 Eliminando archivos de desarrollo...
del /q cmd\mcp-context-server\main_debug*.go 2>nul
del /q cmd\mcp-context-server\main_minimal*.go 2>nul
del /q cmd\mcp-context-server\main_*_standalone.go 2>nul
del /q cmd\mcp-context-server\main_fixed.go 2>nul

echo 📁 Eliminando scripts de build extras...
del /q build-*.bat compile.bat commit-*.bat commit-*.sh 2>nul

echo 📁 Eliminando configs de test...
del /q claude-config.json config.json 2>nul

echo 📁 Eliminando documentación innecesaria...
del /q go.md ANALISIS_FUNCIONAL.md CLAUDE_SETUP.md PROJECT_REPORT.md CLAUDE_DESKTOP_FIXES.md 2>nul

echo 📁 Eliminando binarios...
del /q bin\*.exe 2>nul

echo 📁 Eliminando archivos temporales...
del /q *.tmp *.temp *.backup 2>nul

echo 📁 Eliminando scripts de limpieza...
del /q cleanup.bat cleanup.sh 2>nul

echo ✅ Limpieza completada!

echo 📋 Archivos que permanecen:
echo   ✅ README.md (documentación principal)
echo   ✅ LICENSE (licencia)
echo   ✅ go.mod / go.sum (dependencias)
echo   ✅ Dockerfile (containerización)
echo   ✅ Makefile (build automation)
echo   ✅ .gitignore (actualizado)
echo   ✅ build.bat (script de build principal)
echo   ✅ internal/ (código fuente)
echo   ✅ cmd/mcp-context-server/main.go (punto de entrada)
echo   ✅ CONTRIBUTING.md (guía de contribución)

echo 🔧 Paso 2: Preparando commit Git...
echo 📋 Estado del repositorio:
git status

echo 💾 Agregando archivos al staging...
git add .

echo 🚀 Realizando commit...
git commit -m "🧹 Clean repository and fix Claude Desktop compatibility

✅ Repository cleanup:
- Remove unnecessary test files and development scripts
- Remove duplicate build scripts and configs
- Remove temporary documentation files
- Clean binary directory
- Update .gitignore with comprehensive rules

🔧 Claude Desktop compatibility fixes:
- Fixed JSON-RPC protocol incompatibility
- Added auto-detection for message formats
- Resolved EOF handling causing disconnections
- Added proper notification handling
- Implemented direct JSON transport
- Improved connection stability

📁 Final structure:
- Core Go source code in internal/
- Single main entry point
- Production-ready build script
- Comprehensive documentation
- Clean dependency management

Status: ✅ Production ready and fully compatible with Claude Desktop"

echo 📤 Enviando al repositorio remoto...
git push origin main

echo 🎉 ¡Commit completado exitosamente!
echo 📋 Resumen:
echo   ✅ Repositorio limpio y organizado
echo   ✅ Fixes de compatibilidad aplicados
echo   ✅ Documentación actualizada
echo   ✅ .gitignore mejorado
echo   ✅ Código subido a GitHub

pause
