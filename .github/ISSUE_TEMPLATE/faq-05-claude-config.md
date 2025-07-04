---
name: "❓ FAQ - ¿Cómo configurar correctamente el MCP en Claude Desktop?"
about: Pregunta frecuente sobre la configuración correcta de Claude Desktop
title: "❓ FAQ - ¿Cómo configurar correctamente el MCP en Claude Desktop?"
labels: configuration, claude-desktop, setup, faq
assignees: ''
---

# ❓ Pregunta Frecuente: "¿Cómo configurar correctamente el MCP en Claude Desktop?"

## 🎯 **Problema**
No estoy seguro de si tengo la configuración correcta de Claude Desktop para el MCP Go Context.

## ✅ **Respuesta y Solución**

### **📋 Configuración Paso a Paso**

#### **1. Ubicación del archivo de configuración**

**Windows**:
```
C:\Users\[TuUsuario]\AppData\Roaming\Claude\claude_desktop_config.json
```

**macOS**:
```
~/Library/Application Support/Claude/claude_desktop_config.json
```

**Linux**:
```
~/.config/Claude/claude_desktop_config.json
```

#### **2. Contenido del archivo de configuración**

**Configuración recomendada**:
```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "C:\\MCPs\\clone\\mcp-go-context\\bin\\mcp-context-server.exe",
      "args": ["--transport", "stdio", "--verbose"]
    }
  }
}
```

#### **3. Verificación de rutas**

**Verificar que el ejecutable existe**:
```bash
# Windows
dir "C:\MCPs\clone\mcp-go-context\bin\mcp-context-server.exe"

# Linux/Mac  
ls -la "/path/to/mcp-context-server"
```

**Si no existe, compilar**:
```bash
cd C:\MCPs\clone\mcp-go-context
go build -o bin/mcp-context-server.exe cmd/mcp-context-server/main.go
```

### **🧪 Verificar Configuración**

#### **1. Reiniciar Claude Desktop**
- Cerrar completamente Claude Desktop
- Esperar 5 segundos
- Reabrir

#### **2. Probar conexión**:
```
Usa analyze-project para verificar que el MCP funciona
```

**Resultado esperado**: Análisis detallado del proyecto.

### **⚠️ Problemas Comunes de Configuración**

#### **Problema 1: Ruta incorrecta**
❌ **Error**:
```json
"command": "mcp-context-server.exe"
```

✅ **Correcto**:
```json
"command": "C:\\ruta\\completa\\mcp-context-server.exe"
```

#### **Problema 2: Barras invertidas en Windows**
❌ **Error**:
```json
"command": "C:/MCPs/clone/mcp-go-context/bin/mcp-context-server.exe"
```

✅ **Correcto**:
```json
"command": "C:\\MCPs\\clone\\mcp-go-context\\bin\\mcp-context-server.exe"
```

#### **Problema 3: JSON inválido**
❌ **Error**: Comas extras, comillas mal cerradas
✅ **Verificar**: Usar un validador JSON online

### **🐛 Troubleshooting de Configuración**

#### **A. Claude Desktop no reconoce el MCP**:
1. Verificar que el archivo `claude_desktop_config.json` existe
2. Verificar sintaxis JSON (usar jsonlint.com)
3. Verificar que la ruta del ejecutable es correcta
4. Reiniciar Claude Desktop

#### **B. El MCP se conecta pero no responde**:
1. Añadir flag `--verbose` para logs
2. Verificar permisos del ejecutable
3. Probar ejecutar manualmente el servidor

### **📚 Recursos Adicionales**
- Ver [CLAUDE_SETUP.md](../../CLAUDE_SETUP.md) para configuración detallada
- Ver archivo completo [FAQ-05-claude-config.md](../../docs/faq/FAQ-05-claude-config.md)

---

**¿Funciona la configuración?** Si Claude Desktop reconoce el MCP, marca como resuelto.
