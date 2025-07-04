# 📚 Manual Práctico - MCP Go Context

## 🎯 ¿Por qué no ves beneficios? - Diagnóstico

### **Problema Principal: No usas las herramientas correctamente**

El MCP funciona, pero Claude **no sabe automáticamente** cuándo usar las herramientas. Tienes que **pedírselo explícitamente**.

### **¿Por qué está vacía la carpeta memory?**
La memoria se activa **solo cuando usas la herramienta `remember-conversation`** o cuando las otras herramientas necesitan contexto.

---

## 🚀 **Guía de Uso Paso a Paso**

### **1. Verificar que funciona**

Prueba este comando **literal** en Claude:

```
Usa analyze-project para analizar mi proyecto actual
```

**Deberías ver**: Un análisis completo del proyecto con estadísticas, archivos clave, dependencias, etc.

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

**Si no existe, créalo**:
```json
{
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
```

**Si estos comandos funcionan**, el MCP está correctamente configurado.

---

## 💪 **Potencia Real del MCP**

### **Antes (Sin MCP)**:
- Claude olvida el contexto anterior ❌
- Tienes que reexplicar el proyecto cada vez ❌  
- No tiene acceso a documentación específica ❌
- No puede analizar dependencias ❌

### **Después (Con MCP bien usado)**:
- Claude recuerda decisiones importantes ✅
- Analiza automáticamente la estructura del proyecto ✅
- Accede a documentación relevante ✅
- Entiende dependencies y su impacto ✅
- Da sugerencias basadas en el contexto del proyecto ✅

---

## 🏆 **Tip Final**

**El truco está en ser explícito**. En lugar de:
> "¿Cómo puedo mejorar este código?"

Usa:
> "Usa get-context con query='optimización performance servidor HTTP' y después analiza cómo mejorar este código."

**La diferencia**: El MCP necesita que **le digas qué herramientas usar**, no las activa automáticamente.

Una vez domines esto, verás que Claude se vuelve **muchísimo más útil** porque mantiene contexto y puede acceder a información específica de tu proyecto.
