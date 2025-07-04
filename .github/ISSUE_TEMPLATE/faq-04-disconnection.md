---
name: "❓ FAQ - El MCP se desconecta después de 60 segundos"
about: Pregunta frecuente sobre problemas de desconexión (PROBLEMA RESUELTO)
title: "❓ FAQ - El MCP se desconecta después de 60 segundos"
labels: bug, connection, fixed, stdio, faq
assignees: ''
---

# ❓ Pregunta Frecuente: "El MCP se desconecta después de 60 segundos"

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
   Ejecuta con flag `--verbose` para ver logs detallados.

3. **Reiniciar Claude Desktop**:
   Cerrar completamente y reabrir.

### **📚 Recursos Adicionales**
- [README.md](../../README.md) sección "🔧 Troubleshooting"
- Ver archivo completo [FAQ-04-disconnection.md](../../docs/faq/FAQ-04-disconnection.md)

### **🆘 Si el problema persiste**

Abre un nuevo issue con:
- Versión del sistema operativo
- Logs del servidor con `--verbose`
- Output de `git log --oneline -3`

---

**¿Está funcionando ahora?** Si la conexión es estable, marca como resuelto.
