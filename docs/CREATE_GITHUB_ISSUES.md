# 🎯 Issues FAQ para GitHub - Listo para copiar y pegar

Esta es la colección completa de issues FAQ listos para crear en GitHub. Cada issue incluye título, etiquetas y contenido completo.

---

## Issue #1: FAQ - No veo beneficios del MCP después de un día de uso

**URL para crear**: https://github.com/scopweb/mcp-go-context/issues/new
**Título**: `❓ FAQ - No veo beneficios del MCP después de un día de uso`
**Etiquetas**: `question`, `help-wanted`, `documentation`, `faq`

**Contenido**:
```markdown
# ❓ Pregunta Frecuente: "No veo beneficios del MCP después de un día de uso"

## 🎯 **Problema**
Después de instalar y configurar el MCP Go Context, no noto mejoras significativas en mi flujo de trabajo con Claude Desktop.

## ✅ **Respuesta y Solución**

### **El problema principal: No estás usando las herramientas explícitamente**

El MCP **SÍ funciona**, pero Claude **no activa automáticamente** las herramientas. Tienes que **pedírselo explícitamente**.

### **🧪 Prueba Inmediata**

Copia y pega esto **exactamente** en Claude:

```
Usa analyze-project para analizar mi proyecto actual
```

**Si ves un análisis detallado**, el MCP funciona correctamente.

### **💡 Comandos Esenciales que Debes Usar**

1. **Análisis del proyecto**:
   ```
   Usa analyze-project para ver la estructura del proyecto
   ```

2. **Guardar información importante**:
   ```
   Usa remember-conversation con key="arquitectura-decisión" content="Decidimos usar stdio transport porque Claude Desktop lo requiere" tags=["arquitectura", "decisiones"]
   ```

3. **Obtener contexto inteligente**:
   ```
   Usa get-context con query="debugging server connection issues" para obtener contexto relevante
   ```

4. **Documentación**:
   ```
   Usa fetch-docs con library="gin-gonic/gin" topic="middleware"
   ```

### **🔑 Diferencia Clave**

❌ **Mal**: "¿Cómo puedo mejorar este código?"  
✅ **Bien**: "Usa get-context con query='optimización performance servidor HTTP' y después analiza cómo mejorar este código"

### **📚 Recursos Adicionales**
- Ver [MANUAL.md](./MANUAL.md) para guía completa de uso
- Issue #2: [¿Por qué está vacía mi carpeta memory?](#)
- Issue #3: [¿Cómo usar las herramientas correctamente?](#)

---

**¿Esta respuesta resolvió tu problema?** 👍 Reacciona con 👍 si te ayudó o comenta si necesitas más aclaraciones.
```

---

## Issue #2: FAQ - ¿Por qué está vacía mi carpeta .mcp-context/memory?

**URL para crear**: https://github.com/scopweb/mcp-go-context/issues/new
**Título**: `❓ FAQ - ¿Por qué está vacía mi carpeta .mcp-context/memory?`
**Etiquetas**: `question`, `memory`, `configuration`, `faq`

**Contenido**:
```markdown
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

3. **Ver memoria guardada**:
   El archivo `current.json` contendrá tus memorias en formato JSON.

### **🔧 Verificar Configuración**

Si no funciona, verifica la configuración por defecto en:
`C:\Users\[Usuario]\.mcp-context\config.json`

### **🐛 Troubleshooting**

1. **Permisos**: Verifica que Claude Desktop tenga permisos de escritura
2. **Ruta**: La carpeta se crea automáticamente la primera vez
3. **Logs**: Busca errores en los logs del MCP server

### **📚 Recursos Adicionales**
- Ver [MANUAL.md](./MANUAL.md) sección "Activar la Memoria Persistente"
- Issue #1: [No veo beneficios del MCP](#)
- Issue #3: [¿Cómo usar las herramientas correctamente?](#)

---

**¿Esta respuesta resolvió tu problema?** Si el archivo se creó correctamente, marca como resuelto.
```

---

## Issue #3: FAQ - ¿Cómo usar las herramientas del MCP correctamente?

**URL para crear**: https://github.com/scopweb/mcp-go-context/issues/new
**Título**: `❓ FAQ - ¿Cómo usar las herramientas del MCP correctamente?`
**Etiquetas**: `question`, `documentation`, `tools`, `usage`, `faq`

**Contenido**: [Ver archivo completo FAQ-03-correct-usage.md]

---

## Issue #4: FAQ - El MCP se desconecta después de 60 segundos

**URL para crear**: https://github.com/scopweb/mcp-go-context/issues/new
**Título**: `❓ FAQ - El MCP se desconecta después de 60 segundos`
**Etiquetas**: `bug`, `connection`, `fixed`, `stdio`, `faq`

**Contenido**: [Ver archivo completo FAQ-04-disconnection.md]

---

## Issue #5: FAQ - ¿Cómo configurar correctamente el MCP en Claude Desktop?

**URL para crear**: https://github.com/scopweb/mcp-go-context/issues/new
**Título**: `❓ FAQ - ¿Cómo configurar correctamente el MCP en Claude Desktop?`
**Etiquetas**: `configuration`, `claude-desktop`, `setup`, `faq`

**Contenido**: [Ver archivo completo FAQ-05-claude-config.md]

---

## Issue #6: FAQ - ¿Cuáles son los casos de uso prácticos del MCP?

**URL para crear**: https://github.com/scopweb/mcp-go-context/issues/new
**Título**: `❓ FAQ - ¿Cuáles son los casos de uso prácticos del MCP?`
**Etiquetas**: `documentation`, `use-cases`, `examples`, `workflow`, `faq`

**Contenido**: [Ver archivo completo FAQ-06-use-cases.md]

---

## Issue #7: FAQ - Troubleshooting: Problemas comunes y soluciones

**URL para crear**: https://github.com/scopweb/mcp-go-context/issues/new
**Título**: `❓ FAQ - Troubleshooting: Problemas comunes y soluciones`
**Etiquetas**: `troubleshooting`, `debugging`, `support`, `faq`

**Contenido**: [Ver archivo completo FAQ-07-troubleshooting.md]

---

# 🚀 Instrucciones para Crear los Issues

## Método Manual (Recomendado)

1. **Ve a**: https://github.com/scopweb/mcp-go-context/issues/new
2. **Copia el título** del issue correspondiente
3. **Copia todo el contenido** desde el archivo FAQ correspondiente
4. **Añade las etiquetas** sugeridas
5. **Crea el issue**
6. **Repite** para todos los FAQs

## Método Rápido

1. **Abre todos los archivos FAQ** en `docs/faq/`
2. **Para cada archivo**:
   - Copia todo el contenido
   - Crea nuevo issue en GitHub
   - Pega el contenido
   - Añade etiquetas
3. **Al final**, tendrás 7 issues FAQ perfectamente estructurados

## 🏷️ Etiquetas a Usar

- `faq` - Para todos los issues
- `question` - Para preguntas generales
- `documentation` - Para temas de documentación
- `configuration` - Para problemas de configuración
- `troubleshooting` - Para solución de problemas
- `memory` - Para temas de memoria
- `tools` - Para uso de herramientas
- `bug` - Para problemas técnicos (solo FAQ #4)

¡Una vez creados todos los issues, tendrás un sistema completo de FAQ navegable y buscable en GitHub!
