# ❓ FAQ #4: "El MCP se desconecta después de 60 segundos"

**Etiquetas**: `bug`, `connection`, `fixed`, `stdio`, `faq`

## 🎯 **Problema**
El servidor MCP se desconecta automáticamente después de aproximadamente 60 segundos de funcionamiento.

## ✅ **Respuesta y Solución**

### **🎉 PROBLEMA RESUELTO EN VERSIÓN ACTUAL**

Este problema **ya está solucionado** en la versión actual del MCP Go Context (commit del 2025-07-03).

### **🔧 Qué se solucionó**

**Problemas anteriores**:
- ❌ Server se desconectaba después de ~60 segundos
- ❌ Errores "Content-Length is not valid JSON"  
- ❌ EOF terminando el proceso prematuramente
- ❌ Timeouts durante initialize

**Soluciones implementadas**:
- ✅ **Manejo EOF mejorado**: Reconexión automática con retry logic
- ✅ **Protocolo JSON-RPC directo**: Compatible con Claude Desktop
- ✅ **Auto-detección de formato**: Maneja tanto JSON directo como headers
- ✅ **Manejo de notificaciones**: Soporte para `notifications/cancelled`

### **📁 Archivos modificados**
- `internal/transport/stdio.go` - Manejo EOF y protocolo mejorado
- `internal/server/server.go` - JSON-RPC protocol handling
- README.md actualizado con status "FIXED"

### **🧪 Verificar que está solucionado**

1. **Compilar la versión actual**:
   ```bash
   cd C:\MCPs\clone\mcp-go-context
   go build -o bin/mcp-context-server.exe cmd/mcp-context-server/main.go
   ```

2. **Actualizar configuración de Claude Desktop**:
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

3. **Probar estabilidad**:
   ```
   Usa analyze-project y espera 2-3 minutos, luego usa get-context con query="test stability"
   ```

   **Resultado esperado**: No debería desconectarse.

### **🐛 Si aún tienes el problema**

#### **Diagnóstico**:

1. **Verificar versión**:
   ```bash
   git log --oneline -5
   ```
   Deberías ver commits recientes sobre "stdio fixes" o "protocol improvements".

2. **Logs del servidor**:
   Ejecuta con flag `--verbose` para ver logs detallados:
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

3. **Logs de Claude Desktop**:
   - Windows: `%APPDATA%\Claude\logs\`
   - macOS: `~/Library/Logs/Claude/`

#### **Soluciones adicionales**:

**A. Usar binario pre-compilado**:
```bash
# Usar el binario ya compilado con fixes
"command": "C:\\MCPs\\clone\\mcp-go-context\\bin\\mcp-context-server-ultrafixed.exe"
```

**B. Reiniciar Claude Desktop**:
```bash
# Cerrar completamente Claude Desktop
# Reabrir y probar
```

**C. Verificar permisos**:
```bash
# Asegurar que el ejecutable tiene permisos
chmod +x mcp-context-server.exe  # Linux/Mac
```

### **🔮 Versiones anteriores afectadas**

Si tienes una versión anterior al 2025-07-03, actualiza:

```bash
git pull origin master
go build -o bin/mcp-context-server.exe cmd/mcp-context-server/main.go
```

### **📚 Recursos Adicionales**
- [README.md](../../README.md) sección "🔧 Troubleshooting"
- FAQ #5: [Configuración correcta en Claude Desktop](./FAQ-05-claude-config.md)
- FAQ #7: [Troubleshooting general](./FAQ-07-troubleshooting.md)

### **🆘 Si el problema persiste**

Abre un nuevo issue con:
- Versión del sistema operativo
- Logs del servidor con `--verbose`
- Logs de Claude Desktop
- Output de `git log --oneline -3`

---

**¿Está funcionando ahora?** Si la conexión es estable, marca como resuelto.
