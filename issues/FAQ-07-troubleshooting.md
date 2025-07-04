# ❓ FAQ #7: "Troubleshooting: Problemas comunes y soluciones"

**Etiquetas**: `troubleshooting`, `debugging`, `support`, `faq`

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

3. **Verificar configuración de memoria**:
   ```json
   {
     "memory": {
       "enabled": true,
       "persistent": true,
       "storagePath": "C:\\Users\\[Usuario]\\.mcp-context\\memory.json"
     }
   }
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

4. **Verificar dependencias**:
   ```bash
   go mod verify
   ```

---

### **⚡ Problemas de Rendimiento**

#### **Problema**: Respuestas lentas del MCP
**Síntomas**: Timeouts o respuestas que tardan mucho

**Soluciones**:
1. **Limitar tokens en consultas**:
   ```
   Usa get-context con query="..." maxTokens=3000
   ```

2. **Usar queries específicas**:
   ```
   # ❌ Malo
   Usa get-context con query="help"
   
   # ✅ Bueno
   Usa get-context con query="HTTP middleware authentication"
   ```

3. **Limpiar cache si es necesario**:
   ```bash
   rm -rf "%USERPROFILE%\.mcp-context\cache"  # Windows
   rm -rf "~/.mcp-context/cache"              # Linux/Mac
   ```

---

### **🔍 Problemas de Análisis**

#### **Problema**: analyze-project no encuentra archivos
**Síntomas**: Análisis vacío o incompleto

**Soluciones**:
1. **Verificar directorio de trabajo**:
   ```
   Usa analyze-project con path="C:\ruta\completa\proyecto"
   ```

2. **Verificar permisos de lectura**:
   ```bash
   # Verificar que Claude Desktop puede leer el directorio
   ls -la proyecto/  # Ver permisos
   ```

3. **Verificar patrones de exclusión**:
   Por defecto se excluyen: `.git`, `node_modules`, `vendor`, `bin`, `dist`

---

### **📚 Problemas con fetch-docs**

#### **Problema**: No encuentra documentación
**Síntomas**: "No documentation available"

**Soluciones**:
1. **Usar nombres exactos de librerías**:
   ```
   # ✅ Correcto
   Usa fetch-docs con library="gin-gonic/gin"
   
   # ❌ Incorrecto
   Usa fetch-docs con library="gin"
   ```

2. **Verificar conectividad a Internet**:
   ```bash
   ping context7.com
   ```

3. **Usar alternativas locales**:
   ```
   Usa get-context con query="gin framework documentation"
   ```

---

### **🐛 Problemas Específicos por SO**

#### **Windows**
1. **Antivirus bloqueando el ejecutable**:
   - Añadir excepción en Windows Defender
   - Verificar que no está en cuarentena

2. **Caracteres especiales en rutas**:
   ```json
   # Usar barras dobles
   "command": "C:\\MCPs\\clone\\mcp-go-context\\bin\\mcp-context-server.exe"
   ```

#### **macOS**
1. **Gatekeeper bloqueando ejecutable**:
   ```bash
   xattr -d com.apple.quarantine mcp-context-server
   ```

2. **Permisos de ejecución**:
   ```bash
   chmod +x mcp-context-server
   ```

#### **Linux**
1. **Dependencias faltantes**:
   ```bash
   ldd mcp-context-server  # Verificar dependencias
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

#### **4. Revisar logs**:
**Claude Desktop logs**:
- Windows: `%APPDATA%\Claude\logs\`
- macOS: `~/Library/Logs/Claude/`
- Linux: `~/.config/Claude/logs/`

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
- FAQ #4: [El MCP se desconecta](./FAQ-04-disconnection.md)
- FAQ #5: [Configuración en Claude Desktop](./FAQ-05-claude-config.md)
- [MANUAL.md](../../MANUAL.md) para uso detallado

---

**¿Solucionó esto tu problema?** Si no, por favor abre un issue con los detalles arriba.
