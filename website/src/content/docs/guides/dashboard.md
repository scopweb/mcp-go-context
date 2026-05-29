---
title: Dashboard y API HTTP
description: Cómo usar la interfaz web y los endpoints HTTP para gestionar memoria.
---

El servidor incluye un dashboard web completo cuando lo ejecutas con transporte `http` o `sse`.

## Cómo activarlo

```bash
bin\mcp-context-server.exe --transport http
```

Por defecto escucha en el puerto 3000.

## Qué puedes hacer en el Dashboard

- Ver todas tus memorias con filtros por proyecto, tipo de decisión y búsqueda.
- Ver el contador de memorias **promovidas**.
- Filtrar solo memorias promovidas.
- Ver la sección **"Suggested for Promotion"** con las mejores candidatas según la heurística.
- Promover memorias con un solo clic (botón "Promote").
- Refrescar sugerencias manualmente.

Esta interfaz es especialmente útil para revisar el estado de tu memoria operativa sin tener que usar el chat.

## Endpoints HTTP más útiles

| Endpoint                    | Método | Uso principal |
|----------------------------|--------|---------------|
| `/api/suggestions`         | GET    | Obtener candidatos para promover (equivalente a `suggest-promotions`) |
| `/api/memories`            | GET    | Listar y filtrar memorias |
| `/api/memories/{key}`      | POST   | Promover una memoria concreta |
| `/api/project-summary`     | GET    | Resumen ligero del proyecto + estado de memoria |
| `/api/quick-context`       | GET    | Contexto técnico bajo demanda |

### Ejemplo: Obtener sugerencias de promoción

```bash
curl "http://localhost:3000/api/suggestions?limit=6"
```

### Ejemplo: Promover una memoria vía API

```bash
curl -X POST "http://localhost:3000/api/memories/mi-decision-importante"
```

## Cuándo usar el Dashboard vs herramientas MCP

- **Usa el Dashboard** cuando quieras revisar visualmente el estado de tu memoria o hacer limpieza/promociones masivas.
- **Usa las herramientas MCP** (`suggest-promotions`, `promote-memory`, etc.) cuando estés dentro de una sesión de trabajo con tu asistente.

Ambos mundos están sincronizados: lo que promotes desde el dashboard aparece inmediatamente en las herramientas, y viceversa.