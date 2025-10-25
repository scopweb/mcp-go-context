# Configuración para Claude Desktop

## Instalación

1. Ejecuta `build.bat` para compilar el servidor
2. Copia la configuración del servidor a tu claude_desktop_config.json

## Configuración para claude_desktop_config.json

### Opción 1: Ejecutable local
```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "C:\\MCPs\\clone\\mcp-go-context\\bin\\mcp-go-context.exe",
      "args": ["--transport", "stdio"]
    }
  }
}
```

### Opción 2: Con archivo de configuración
```json
{
  "mcpServers": {
    "mcp-go-context": {
      "command": "C:\\MCPs\\clone\\mcp-go-context\\bin\\mcp-go-context.exe",
      "args": ["--transport", "stdio", "--config", "C:\\MCPs\\clone\\mcp-go-context\\config.json"]
    }
  }
}
```

## Herramientas disponibles

Una vez configurado, tendrás acceso a estas herramientas en Claude:

- **analyze-project**: Analiza la estructura del proyecto
- **get-context**: Obtiene contexto relevante para tu consulta
- **fetch-docs**: Obtiene documentación de librerías
- **remember-conversation**: Guarda información importante
- **dependency-analysis**: Analiza las dependencias del proyecto

## Uso

Simplemente menciona en Claude:
- "Analiza este proyecto"
- "Dame contexto sobre autenticación"
- "Recuerda que usamos JWT para auth"
- "¿Qué dependencias tenemos?"

El servidor automáticamente proporcionará contexto enriquecido basado en tu proyecto local.
