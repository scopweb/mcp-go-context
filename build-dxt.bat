@echo off
echo 🏗️ Building MCP Go Context Desktop Extension (.dxt)

cd /d C:\MCPs\clone\mcp-go-context

echo 📦 Creating dxt build directory...
if exist dxt-build rmdir /s /q dxt-build
mkdir dxt-build

echo 🔨 Building Go binary...
go build -ldflags "-X main.version=2.0.0 -X main.commit=%DATE:~10,4%%DATE:~4,2%%DATE:~7,2%" -o dxt-build\bin\mcp-context-server.exe cmd\mcp-context-server\main.go

if %errorlevel% neq 0 (
    echo ❌ Build failed!
    exit /b 1
)

echo 📋 Copying extension files...
copy dxt\manifest.json dxt-build\
copy dxt\README.md dxt-build\
copy dxt\package.json dxt-build\
copy dxt\CHANGELOG.md dxt-build\
copy LICENSE dxt-build\

echo 🎨 Creating placeholder icon...
if not exist dxt-build\screenshots mkdir dxt-build\screenshots
echo. > dxt-build\icon.png
echo. > dxt-build\screenshots\project-analysis.png
echo. > dxt-build\screenshots\context-retrieval.png
echo. > dxt-build\screenshots\memory-management.png

echo 📁 DXT Package Structure:
dir /b dxt-build

echo 🎯 Creating .dxt archive...
cd dxt-build
powershell -Command "Compress-Archive -Path * -DestinationPath ..\mcp-go-context.dxt -Force"
cd ..

if exist mcp-go-context.dxt (
    echo ✅ Desktop Extension created: mcp-go-context.dxt
    echo 📏 File size: 
    for %%A in (mcp-go-context.dxt) do echo %%~zA bytes
    echo.
    echo 🚀 Installation:
    echo   1. Drag mcp-go-context.dxt into Claude Desktop
    echo   2. Configure optional settings
    echo   3. Start using MCP tools!
    echo.
) else (
    echo ❌ Failed to create .dxt file!
    exit /b 1
)

echo 🧹 Cleaning up build directory...
rmdir /s /q dxt-build

echo ✨ Build complete!
pause