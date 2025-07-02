@echo off
echo 🔨 Testing compilation...

cd C:\MCPs\clone\mcp-go-context

echo 📦 Running go mod tidy...
go mod tidy

echo 🏗️ Building...
go build -o test-build.exe test-compilation.go

if %errorlevel% == 0 (
    echo ✅ Build successful!
    echo 🧪 Running test...
    test-build.exe
    del test-build.exe
) else (
    echo ❌ Build failed!
)

pause
