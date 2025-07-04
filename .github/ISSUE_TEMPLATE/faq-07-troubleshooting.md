---
name: "❓ FAQ - Troubleshooting: Problemas comunes y soluciones"
about: Pregunta frecuente sobre troubleshooting y resolución de problemas
title: "❓ FAQ - Troubleshooting: Problemas comunes y soluciones"
labels: troubleshooting, debugging, support, faq
assignees: ''
---

# ❓ Pregunta Frecuente: "Troubleshooting: Problemas comunes y soluciones"

## 🎯 **Problema**
Lista completa de problemas comunes del MCP Go Context y sus soluciones.

## ✅ **Guía de Troubleshooting**

### **🔧 Problemas de Conexión**

#### **Problema**: Claude Desktop no reconoce el MCP
**Síntomas**: No aparecen herramientas MCP disponibles

**Soluciones**:
1. **Verificar configuración**:
   ```bash
   # Verificar que existe el archivo
   cat "%APPDATA%\Claude\claude_desktop_config.json"  # Windows
   cat "~/Library/Application Support/Claude/claude_desktop_config.json"  # macOS
   ```

2. **Validar JSON**:
   ```bash
   # Usar validador online: jsonlint.com
   # O usar jq si está instalado
   jq . claude_desktop_config.json
   ```

3. **Reiniciar Claude Desktop**:
   - Cerrar completamente (incluyendo tray)
   - Esperar 5 segundos
   - Reabrir

4. **Verificar ruta del ejecutable**:
   ```bash
   dir "C:\MCPs\clone\mcp-go-context\bin\mcp-context-server.exe"
   ```

---

#### **Problema**: MCP se conecta pero no responde
**Síntomas**: Herramientas aparecen pero dan timeout o error

**Soluciones**:
1. **Añadir logs verbosos**:
   ```json
   {
     "mcpServers": {
       "mcp-go-context": {
         "command": "...",
         "args": ["--transport", "stdio", "--verbose"]
       }
     }
   }
   ```

2. **Verificar permisos**:
   ```bash
   # Windows: Ejecutar como administrador si es necesario
   # Linux/Mac: 
   chmod +x mcp-context-server
   ```

3. **Probar manualmente**:
   ```bash
   # Ejecutar el servidor directamente
   ./mcp-context-server.exe --transport stdio --verbose
   ```

---

### **💾 Problemas de Memoria**

#### **Problema**: No se crean archivos en .mcp-context
**Síntomas**: Carpeta vacía o inexistente

**Soluciones**:
1. **Usar herramientas explícitamente**:
   ```
   Usa remember-conversation con key="test" content="prueba" tags=["test"]
   ```

2. **Verificar permisos de escritura**:
   ```bash
   # Crear directorio manualmente si es necesario
   mkdir "%USERPROFILE%\.mcp-context"  # Windows
   mkdir "~/.mcp-context"              # Linux/Mac
   ```

---

### **🛠️ Problemas de Compilación**

#### **Problema**: Error al compilar el servidor
**Síntomas**: `go build` falla

**Soluciones**:
1. **Verificar versión de Go**:
   ```bash
   go version
   # Necesitas Go 1.21 o superior
   ```

2. **Limpiar módulos**:
   ```bash
   go mod tidy
   go mod download
   ```

3. **Compilar desde directorio correcto**:
   ```bash
   cd C:\MCPs\clone\mcp-go-context
   go build -o bin/mcp-context-server.exe cmd/mcp-context-server/main.go
   ```

---

### **📋 Diagnóstico Sistemático**

Si tienes un problema no listado, sigue estos pasos:

#### **1. Verificar lo básico**:
```bash
# 1. Ejecutable existe
ls -la mcp-context-server.exe

# 2. Es ejecutable
./mcp-context-server.exe --help

# 3. JSON válido
jq . claude_desktop_config.json

# 4. Claude Desktop reiniciado
```

#### **2. Habilitar logs verbosos**:
```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "...",
      "args": ["--transport", "stdio", "--verbose"],
      "env": {
        "MCP_DEBUG": "1"
      }
    }
  }
}
```

#### **3. Probar comandos básicos**:
```
1. Usa analyze-project
2. Usa remember-conversation con key="test" content="test" tags=["test"]
3. Usa get-context con query="test"
```

### **🆘 Solicitar Ayuda**

Si el problema persiste, abre un issue con:

```
**Sistema Operativo**: Windows 11 / macOS Ventura / Ubuntu 22.04
**Versión Go**: go version
**Versión MCP**: git log --oneline -3
**Configuración**: [tu claude_desktop_config.json]
**Logs**: [logs relevantes]
**Pasos para reproducir**: [lista detallada]
**Comportamiento esperado**: [qué debería pasar]
**Comportamiento actual**: [qué pasa realmente]
```

### **📚 Recursos Adicionales**
- [README.md](../../README.md) sección troubleshooting
- Ver archivo completo [FAQ-07-troubleshooting.md](../../docs/faq/FAQ-07-troubleshooting.md)

---

**¿Solucionó esto tu problema?** Si no, por favor abre un issue con los detalles arriba.
