---
title: Guía de introducción
description: Primeros pasos con mcp-go-context.
---

Esta guía te ayuda a comenzar a usar el servidor MCP de contexto para gestionar contexto técnico en tus proyectos.

## Requisitos previos

- Go 1.26.2 o superior
- Un proyecto de código existente
- (Opcional) Claude Desktop o Claude Code para uso como servidor MCP

## Instalación

### Desde código fuente

```bash
git clone https://github.com/scopweb/mcp-go-context
cd mcp-go-context
go build -o bin/mcp-context-server ./cmd/mcp-context-server
```

### Configuración en Claude Desktop

Agrega esto a tu configuración de Claude Desktop:

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

## Uso rápido

### Análisis de proyecto

```
Use analyze-project to get a quick overview of the project structure and dependencies.
```

### Buscar decisiones pasadas

```
Use get-decisions with a keyword filter to find why a technical choice was made.
```

### Contexto para una tarea

```
Use get-context with a query like "database migrations" or "authentication flow".
```

### Guardar una decisión técnica (flujo moderno)

```
1. Usa save-decision con reason y alternatives cuando sea posible.
2. Al terminar la sesión ejecuta suggest-promotions.
3. Promueve solo las recomendaciones que realmente importan usando promote-memory.
```

### Analizar dependencias

```
Use dependency-analysis to see all direct and indirect dependencies with recommendations.
```

### Traer documentación

```
Use fetch-docs with library name and optional topic. Falls back to local docs if offline.
```

## Transporte HTTP para consumidores externos

Cuando ejecutas el servidor con transporte HTTP o SSE, puedes hacer enrichment de contexto desde otros servicios:

```bash
# Resumen de proyecto para inicio de sesión
curl "http://localhost:3000/api/project-summary?path=.&depth=2"

# Contexto técnico bajo demanda
curl "http://localhost:3000/api/quick-context?query=database+migrations&maxTokens=2000"
```

Para activar HTTP, usa `--transport http` en lugar de `stdio`.

## Próximos pasos

- Lee **[Flujo de Memoria](/guides/memory-convergence/)** para entender el flujo actual recomendado.
- Lee **[Buenas Prácticas](/guides/best-practices/)** — es la guía más importante para mantener memoria de calidad.
- Explora el **[Dashboard](/guides/dashboard/)** si prefieres trabajar visualmente.
- Consulta la **[Referencia de herramientas](/reference/tools/)**.