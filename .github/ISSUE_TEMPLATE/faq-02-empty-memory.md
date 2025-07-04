---
name: "❓ FAQ - ¿Por qué está vacía mi carpeta .mcp-context/memory?"
about: Pregunta frecuente sobre la carpeta de memoria vacía
title: "❓ FAQ - ¿Por qué está vacía mi carpeta .mcp-context/memory?"
labels: question, memory, configuration, faq
assignees: ''
---

# ❓ Pregunta Frecuente: "¿Por qué está vacía mi carpeta .mcp-context/memory?"

## 🎯 **Problema**
He configurado el MCP pero no veo archivos en `C:\Users\[Usuario]\.mcp-context\memory` y el archivo `memory.json` no se crea.

## ✅ **Respuesta y Solución**

### **💾 La memoria se activa SOLO cuando la usas explícitamente**

El sistema de memoria **NO guarda automáticamente**. Debes usar la herramienta `remember-conversation` para que se creen archivos.

### **🧪 Prueba Inmediata**

Ejecuta esto en Claude:

```
Usa remember-conversation con key="test-memory" content="Esto es una prueba del sistema de memoria del MCP" tags=["test", "memoria"]
```

**Resultado esperado**: 
- ✅ Mensaje de confirmación
- 📁 Archivo `current.json` creado en `C:\Users\[Usuario]\.mcp-context\`

### **📂 Estructura de Memoria**

```
C:\Users\[Usuario]\.mcp-context\
├── current.json          # Sesión actual con tus memorias
├── config.json          # Configuración (opcional)
└── cache/               # Cache de análisis (opcional)
    └── ...
```

### **💡 Cómo Funciona la Memoria**

1. **Guardar memoria**:
   ```
   Usa remember-conversation con key="proyecto-setup" content="Este proyecto es un servidor MCP que gestiona contexto para Claude Desktop" tags=["proyecto", "arquitectura"]
   ```

2. **Buscar memoria**:
   ```
   Usa get-context con query="proyecto setup" para recuperar información guardada
   ```

### **🔧 Verificar Configuración**

Si no funciona, verifica la configuración por defecto en:
`C:\Users\[Usuario]\.mcp-context\config.json`

### **🐛 Troubleshooting**

1. **Permisos**: Verifica que Claude Desktop tenga permisos de escritura
2. **Ruta**: La carpeta se crea automáticamente la primera vez
3. **Logs**: Busca errores en los logs del MCP server

### **📚 Recursos Adicionales**
- Ver [MANUAL.md](../../MANUAL.md) sección "Activar la Memoria Persistente"
- Ver otros FAQs en [docs/faq/](../../docs/faq/)

---

**¿Esta respuesta resolvió tu problema?** Si el archivo se creó correctamente, marca como resuelto.
