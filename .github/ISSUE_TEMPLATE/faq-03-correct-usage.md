---
name: "❓ FAQ - ¿Cómo usar las herramientas del MCP correctamente?"
about: Pregunta frecuente sobre el uso correcto de las herramientas MCP
title: "❓ FAQ - ¿Cómo usar las herramientas del MCP correctamente?"
labels: question, documentation, tools, usage, faq
assignees: ''
---

# ❓ Pregunta Frecuente: "¿Cómo usar las herramientas del MCP correctamente?"

## 🎯 **Problema**
No entiendo cuándo y cómo usar cada herramienta del MCP Go Context de manera efectiva.

## ✅ **Respuesta y Solución**

### **🛠️ Herramientas Disponibles**

| Herramienta | Cuándo Usar | Ejemplo |
|-------------|-------------|---------|
| `analyze-project` | Al empezar o cuando necesitas visión general | `Usa analyze-project` |
| `get-context` | Para obtener información específica | `Usa get-context con query="error handling"` |
| `remember-conversation` | Para guardar decisiones importantes | `Usa remember-conversation con key="..."` |
| `fetch-docs` | Para documentación de librerías | `Usa fetch-docs con library="gin"` |
| `dependency-analysis` | Para revisar dependencias | `Usa dependency-analysis` |

### **📝 Casos de Uso Prácticos**

#### **🔍 1. Análisis Inicial de Proyecto**
```
Usa analyze-project para entender la estructura del proyecto
```
**Cuándo**: Primera vez trabajando con un proyecto, o después de cambios importantes.

#### **💭 2. Guardar Decisiones Arquitectónicas**
```
Usa remember-conversation con key="arquitectura-stdio" content="Elegimos stdio transport porque Claude Desktop solo soporta eso. HTTP será para futuras versiones" tags=["arquitectura", "decisiones", "transport"]
```
**Cuándo**: Después de tomar decisiones importantes que quieres recordar.

#### **🎯 3. Obtener Contexto para Debugging**
```
Usa get-context con query="connection timeout issues stdio transport" para entender problemas de conexión
```
**Cuándo**: Cuando trabajas en un problema específico y necesitas contexto relevante.

### **💡 Workflows Recomendados**

#### **📋 Workflow: Empezar un Nuevo Proyecto**
1. `Usa analyze-project`
2. `Usa dependency-analysis`
3. `Usa remember-conversation` (guardar objetivos y estructura)

#### **🐛 Workflow: Debugging**
1. `Usa get-context con query="[descripción del problema]"`
2. Analizar y resolver
3. `Usa remember-conversation` (guardar solución)

### **⚠️ Errores Comunes**

❌ **Error**: Usar queries genéricas
```
Usa get-context con query="help"
```

✅ **Correcto**: Ser específico
```
Usa get-context con query="HTTP server implementation gin framework middleware"
```

### **📚 Recursos Adicionales**
- Ver [MANUAL.md](../../MANUAL.md) para workflows completos
- Ver archivo completo [FAQ-03-correct-usage.md](../../docs/faq/FAQ-03-correct-usage.md)

---

**¿Esta respuesta resolvió tu problema?** ¿Qué workflow te resulta más útil?
