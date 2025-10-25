# 📚 Manual Práctico - MCP Go Context 2.0

## 🚀 **Nueva Versión 2025: MCP 2025 + Desktop Extensions**

### **✨ Novedades Principales:**
- 🎯 **Desktop Extensions (.dxt)** - Instalación con un clic
- 🔒 **JWT Authentication** - Seguridad moderna  
- 🛡️ **CORS Configurables** - Protección de orígenes
- 📡 **Streamable HTTP** - Protocolo MCP 2025-03-26
- ⚙️ **11 Herramientas** disponibles con nuevas capacidades

---

## 🎯 ¿Por qué no ves beneficios? - Diagnóstico

### **Problema Principal: No usas las herramientas correctamente**

El MCP funciona, pero Claude **no sabe automáticamente** cuándo usar las herramientas. Tienes que **pedírselo explícitamente**.

### **¿Por qué está vacía la carpeta memory?**
La memoria se activa **solo cuando usas la herramienta `remember-conversation`** o cuando las otras herramientas necesitan contexto.

---

## 🚀 **Guía de Uso Paso a Paso**

### **🆕 0. Instalación Recomendada (Nueva)**

**Método 1: Desktop Extension (Más Fácil)**
1. Descarga `mcp-go-context.dxt`
2. Arrastra el archivo a Claude Desktop
3. Configura opciones opcionales via UI
4. ¡Listo para usar!

**Método 2: Configuración Tradicional (Igual que antes)**
```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "C:\\path\\to\\mcp-context-server.exe",
      "args": ["--transport", "stdio", "--verbose"]
    }
  }
}
```

### **1. Verificar que funciona**

Prueba este comando **literal** en Claude:

```
Usa analyze-project para analizar mi proyecto actual
```

**Deberías ver**: Un análisis completo del proyecto con estadísticas, archivos clave, dependencias, etc.

### **🆕 1.1 Nuevas Herramientas de Seguridad**

Para generar tokens JWT (desarrollo):
```
Usa auth-generate-token con subject="mi-usuario" para generar un token JWT
```

---

### **2. Activar la Memoria Persistente**

Para que el sistema recuerde cosas importantes:

```
Usa remember-conversation con key="proyecto-principal" content="Este es un servidor MCP en Go que gestiona contexto para Claude Desktop. Los archivos clave son server.go, tools.go y memory/manager.go" tags=["proyecto", "go", "mcp"]
```

**Resultado esperado**: 
- ✅ Mensaje de confirmación
- 📁 Archivo creado en `C:\Users\David\.mcp-context\`

---

### **3. Recuperar Contexto Inteligente**

Cuando trabajas en algo específico:

```
Usa get-context con query="debugging server connection issues" para obtener contexto relevante del proyecto
```

**Esto hace**:
- 🧠 Busca en memoria persistente
- 📄 Analiza archivos relevantes  
- 💡 Sugiere código relacionado

---

### **4. Obtener Documentación**

En lugar de buscar en Google:

```
Usa fetch-docs con library="gin-gonic/gin" topic="middleware" para obtener documentación
```

**Ventaja**: Documentación específica sin salir de Claude.

---

## 💡 **Casos de Uso Prácticos**

### **Caso 1: Debugging**
```
Estoy teniendo problemas de conexión con el servidor MCP. 
Usa get-context con query="connection issues stdio transport" 
Después analiza qué puede estar mal.
```

### **Caso 2: Añadir Nueva Funcionalidad**
```
Quiero añadir una nueva herramienta al MCP.
Usa analyze-project para ver la estructura actual.
Después usa get-context con query="adding new tool registry" files=["internal/tools/tools.go", "internal/tools/registry.go"]
```

### **Caso 3: Guardar Decisiones Importantes**
```
Usa remember-conversation con key="arquitectura-decision-1" content="Decidimos usar stdio transport en lugar de HTTP porque Claude Desktop lo requiere. El problema de disconnection se solucionó con el manejo de EOF en stdio.go línea 45" tags=["arquitectura", "decisiones", "stdio"]
```

### **Caso 4: Código Review**
```
Usa get-context con query="security best practices" para revisar el código.
Después usa dependency-analysis con includeTransitive=true para verificar dependencias.
```

---

## 🔧 **Configuración Avanzada**

### **Verificar Configuración Actual**

Revisa el archivo de configuración por defecto en:
`C:\Users\David\.mcp-context\config.json`

**🆕 Configuración Actualizada (MCP 2025)**:
```json
{
  "transport": {
    "type": "stdio",
    "port": 3000
  },
  "memory": {
    "enabled": true,
    "persistent": true,
    "storagePath": "C:\\Users\\David\\.mcp-context\\memory.json",
    "maxEntries": 1000,
    "sessionTTLDays": 30
  },
  "context": {
    "maxTokens": 15000,
    "autoDetectDeps": true,
    "projectPaths": ["C:\\tu\\proyecto\\actual"]
  },
  "security": {
    "auth": {
      "enabled": false,
      "method": "jwt",
      "expiry": "1h"
    },
    "cors": {
      "enabled": true,
      "origins": ["app://claude-desktop", "https://localhost:3000"]
    }
  }
}
```

### **🆕 Configuración Avanzada HTTP/SSE**

Para uso con aplicaciones web o desarrollo:
```json
{
  "transport": {
    "type": "streamable-http",
    "port": 3000
  },
  "security": {
    "auth": {
      "enabled": true,
      "method": "jwt",
      "secret": "tu-secreto-jwt-aqui",
      "expiry": "1h"
    },
    "cors": {
      "enabled": true,
      "origins": ["https://localhost:3000", "*.tudominio.com"]
    }
  }
}
```

### **Aumentar Límites**

Si trabajas con proyectos grandes:
- `maxTokens`: 20000 (más contexto)
- `maxEntries`: 2000 (más memoria)
- `sessionTTLDays`: 60 (memoria más duradera)

---

## 📋 **Workflow Recomendado**

### **Al empezar un proyecto nuevo:**

1. ```
   Usa analyze-project para entender la estructura
   ```

2. ```
   Usa remember-conversation con key="proyecto-setup" content="[Descripción del proyecto y objetivos]" tags=["setup", "objetivos"]
   ```

3. ```
   Usa dependency-analysis para entender las dependencias
   ```

### **Durante el desarrollo:**

1. **Antes de hacer cambios**:
   ```
   Usa get-context con query="[lo que quieres hacer]" para obtener contexto relevante
   ```

2. **Después de decisiones importantes**:
   ```
   Usa remember-conversation para guardar el razonamiento
   ```

3. **Para documentación**:
   ```
   Usa fetch-docs con library="[librería]" cuando necesites referencias
   ```

---

## 🐛 **Troubleshooting**

### **"No veo archivos en memory/"**
- ✅ **Solución**: Usa `remember-conversation` explícitamente
- El sistema no guarda automáticamente, necesitas pedírselo

### **"Las herramientas no funcionan"**
- ✅ **Verifica**: Que Claude Desktop esté usando el MCP correcto
- ✅ **Comando**: Reinicia Claude Desktop después de cambios

### **"El contexto no es útil"**
- ✅ **Mejora**: Sé más específico en las queries
- ❌ Malo: `get-context con query="help"`
- ✅ Bueno: `get-context con query="error handling in HTTP transport layer"`

### **"Respuestas muy largas"**
- ✅ **Limita**: Usa `maxTokens` en get-context
- ✅ **Ejemplo**: `get-context con query="..." maxTokens=3000`

---

## 🎯 **Comandos de Prueba**

Copia y pega estos **exactamente** para probar:

```
1. Usa analyze-project para ver la estructura del proyecto

2. Usa remember-conversation con key="test-memory" content="Esto es una prueba del sistema de memoria" tags=["test", "memoria"]

3. Usa get-context con query="test memory" para ver si recupera la memoria

4. Usa dependency-analysis para ver las dependencias del proyecto

5. Usa fetch-docs con library="golang" topic="http servers" para obtener documentación

🆕 6. Usa memory-search con query="test" para buscar en memorias guardadas

🆕 7. Usa memory-recent con limit=5 para ver memorias recientes

🆕 8. Usa config-get-project-paths para ver rutas configuradas

🆕 9. Usa auth-generate-token con subject="test-user" para generar token JWT (si tienes JWT habilitado)
```

**Si estos comandos funcionan**, el MCP está correctamente configurado.

---

## 🆕 **Nuevas Capacidades MCP 2025**

### **🚀 Transportes Disponibles**

1. **stdio** (Claude Desktop - Recomendado)
   ```bash
   ./mcp-context-server --transport stdio
   ```
   - ✅ Sin autenticación necesaria
   - ✅ Compatible con Claude Desktop
   - ✅ Configuración tradicional funciona igual

2. **streamable-http** (MCP 2025 - Nuevo)
   ```bash
   ./mcp-context-server --transport streamable-http --port 3000
   ```
   - 🆕 Protocolo híbrido HTTP + SSE
   - 🆕 Comunicación bidireccional
   - 🆕 Session management automático

3. **http** (API tradicional)
   ```bash
   ./mcp-context-server --transport http --port 3000
   ```
   - 🔒 Autenticación JWT opcional
   - 🛡️ CORS configurable
   - 📡 Request-response clásico

### **🔐 Seguridad Mejorada**

- **JWT Authentication**: Tokens seguros con expiración
- **CORS Protection**: Lista blanca de orígenes (no más wildcard `*`)
- **Input Validation**: Validación estricta de parámetros
- **Path Security**: Protección contra directory traversal

### **🛠️ Herramientas Ampliadas (11 Total)**

**Nuevas herramientas de memoria:**
- `memory-get` - Obtener memoria por clave
- `memory-search` - Buscar en memorias por texto/tags  
- `memory-recent` - Memorias recientes
- `memory-clear` - Limpiar todas las memorias

**Herramientas de configuración:**
- `config-get-project-paths` - Ver rutas configuradas
- `auth-generate-token` - Generar tokens JWT

---

## 💪 **Potencia Real del MCP 2.0**

### **Antes (Sin MCP o MCP v1)**:
- Claude olvida el contexto anterior ❌
- Tienes que reexplicar el proyecto cada vez ❌  
- No tiene acceso a documentación específica ❌
- No puede analizar dependencias ❌
- Sin búsqueda en memorias ❌
- Instalación manual compleja ❌

### **Después (MCP 2.0 bien usado)**:
- Claude recuerda decisiones importantes ✅
- Analiza automáticamente la estructura del proyecto ✅
- Accede a documentación relevante ✅
- Entiende dependencies y su impacto ✅
- Da sugerencias basadas en el contexto del proyecto ✅
- 🆕 Búsqueda inteligente en memorias históricas ✅
- 🆕 Instalación con un clic via Desktop Extension ✅
- 🆕 Seguridad enterprise-grade ✅
- 🆕 Protocolo MCP 2025 compliant ✅

---

## 🔄 **Migración desde Versión Anterior**

### **¿Tienes MCP v1.x instalado?**

**✅ Buenas noticias: No necesitas cambiar nada**

Tu configuración actual en `claude_desktop_config.json` sigue funcionando **exactamente igual**:

```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "C:\\path\\to\\mcp-context-server.exe",
      "args": ["--transport", "stdio", "--verbose"]  
    }
  }
}
```

### **🆕 Opción de Mejora (Opcional)**

1. **Descarga** `mcp-go-context.dxt`
2. **Arrastra** a Claude Desktop  
3. **Desinstala** configuración manual anterior
4. **Disfruta** instalación simplificada

### **🔧 Variables de Entorno Nuevas (Opcionales)**

```bash
# Para autenticación JWT (solo HTTP/SSE)
export MCP_JWT_SECRET="tu-secreto-aqui"

# Para configuración personalizada  
export MCP_CONFIG_PATH="path/to/config.json"
```

**Nota**: Solo necesarias si usas transportes HTTP/SSE avanzados.

---

## 🏆 **Tip Final**

**El truco está en ser explícito**. En lugar de:
> "¿Cómo puedo mejorar este código?"

Usa:
> "Usa get-context con query='optimización performance servidor HTTP' y después analiza cómo mejorar este código."

**La diferencia**: El MCP necesita que **le digas qué herramientas usar**, no las activa automáticamente.

Una vez domines esto, verás que Claude se vuelve **muchísimo más útil** porque mantiene contexto y puede acceder a información específica de tu proyecto.
