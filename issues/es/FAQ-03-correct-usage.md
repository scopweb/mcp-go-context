# ❓ FAQ #3: "¿Cómo usar las herramientas del MCP correctamente?"

**Etiquetas**: `question`, `documentation`, `tools`, `usage`, `faq`

## 🎯 **Problema**
No entiendo cuándo y cómo usar cada herramienta del MCP Go Context de manera efectiva.

## ✅ **Respuesta y Solución**

### **🛠️ Herramientas Disponibles**

El MCP Go Context proporciona 5 herramientas principales:

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

#### **📚 4. Buscar Documentación**
```
Usa fetch-docs con library="golang" topic="JSON-RPC" para obtener información sobre implementación
```
**Cuándo**: Necesitas referencias técnicas específicas.

#### **🔍 5. Revisar Dependencias**
```
Usa dependency-analysis con includeTransitive=true para ver todas las dependencias
```
**Cuándo**: Antes de actualizaciones o para entender el ecosistema del proyecto.

### **💡 Workflows Recomendados**

#### **📋 Workflow: Empezar un Nuevo Proyecto**
1. `Usa analyze-project`
2. `Usa dependency-analysis`
3. `Usa remember-conversation` (guardar objetivos y estructura)

#### **🐛 Workflow: Debugging**
1. `Usa get-context con query="[descripción del problema]"`
2. Analizar y resolver
3. `Usa remember-conversation` (guardar solución)

#### **⚡ Workflow: Añadir Nueva Funcionalidad**
1. `Usa get-context con query="[funcionalidad relacionada]"`
2. `Usa fetch-docs con library="[librería necesaria]"`
3. Implementar
4. `Usa remember-conversation` (documentar cambios)

### **🔄 Combinando Herramientas**

**Ejemplo complejo**:
```
Quiero añadir logging al servidor MCP.

1. Usa get-context con query="logging implementation server.go" 
2. Usa fetch-docs con library="logrus" topic="structured logging"
3. [Después de implementar] Usa remember-conversation con key="logging-implementation" content="Agregamos logrus con niveles INFO/ERROR. Configuración en config.go línea 45. Logs van a stderr para no interferir con stdio transport" tags=["logging", "implementación", "server"]
```

### **⚠️ Errores Comunes**

❌ **Error**: Usar queries genéricas
```
Usa get-context con query="help"
```

✅ **Correcto**: Ser específico
```
Usa get-context con query="HTTP server implementation gin framework middleware"
```

❌ **Error**: No usar tags en memoria
```
Usa remember-conversation con key="fix" content="Arreglé un bug"
```

✅ **Correcto**: Tags descriptivos
```
Usa remember-conversation con key="fix-stdio-eof" content="Solucioné EOF handling en stdio.go añadiendo retry logic" tags=["bug", "stdio", "eof", "transport"]
```

### **📚 Recursos Adicionales**
- Ver [MANUAL.md](../../MANUAL.md) para workflows completos
- FAQ #1: [No veo beneficios del MCP](./FAQ-01-no-benefits.md)
- FAQ #2: [¿Por qué está vacía mi carpeta memory?](./FAQ-02-empty-memory.md)
- FAQ #6: [Casos de uso prácticos](./FAQ-06-use-cases.md)

---

**¿Esta respuesta resolvió tu problema?** ¿Qué workflow te resulta más útil?
