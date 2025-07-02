#!/bin/bash

# Script de compilación simple
echo "🔨 Compilando MCP Context Server..."

# Ir al directorio del proyecto
cd "$(dirname "$0")"

# Ejecutar go mod tidy
echo "📦 Ejecutando go mod tidy..."
go mod tidy

# Compilar
echo "🏗️  Compilando..."
go build -v -o bin/mcp-context-server ./cmd/mcp-context-server

if [ $? -eq 0 ]; then
    echo "✅ Compilación exitosa!"
    echo "📍 Ejecutable: bin/mcp-context-server"
    
    # Mostrar información del archivo
    if [ -f "bin/mcp-context-server" ]; then
        ls -la bin/mcp-context-server
        echo ""
        echo "🧪 Probando el ejecutable..."
        ./bin/mcp-context-server --version 2>/dev/null || echo "Versión no disponible"
    fi
else
    echo "❌ Error en la compilación"
    exit 1
fi
