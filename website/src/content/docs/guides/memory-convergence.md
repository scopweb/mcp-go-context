---
title: Convergencia de memoria
description: Cómo trabajar junto con SessionMemory de Claude Code.
---

## El modelo

Este servidor está diseñado para trabajar junto con SessionMemory de Claude Code, no para reemplazarlo.

| SessionMemory (summary.md) | mcp-go-context |
|---------------------------|-------------------------------|
| Compaction de nivel de sesión | Memorias persistentes de alto valor |
| Contexto transitorio | Decisiones, fixes, convenciones |
| Auto-generado por Claude | Promovido explícitamente por el usuario |

## Qué va en cada uno

### SessionMemory (summary.md)

- Resumen de la sesión actual
- Contexto transitorio de la conversación
- Churn natural de información temporal
- Decisiones que aún no están consolidadas

### mcp-go-context

- **Decisiones técnicas confirmadas**: arquitectura, elección de herramientas
- **Fixes relevantes**: bugs importantes y su solución
- **Convenciones del repo**: patrones establecidos y reglas
- **Contexto de debugging**: información que ayuda en futuras sesiones

## Flujo de promoción recomendado (2026)

El flujo moderno y más efectivo es el siguiente:

```
Durante el trabajo
       ↓
save-decision (con reason + alternatives cuando sea posible)
       ↓
Al final de sesión compleja o cambio de contexto
       ↓
suggest-promotions
       ↓
Revisas las sugerencias
       ↓
promote-memory (solo lo que realmente importa)
       ↓
Sesiones futuras → get-context y search-memory ya lo traen
```

## Ejemplos prácticos reales

### Ejemplo 1: Decisión de arquitectura

Durante una sesión decides cambiar la estrategia de autenticación:

```text
save-decision
  key="auth-jwt-refresh-2026"
  decisionType="architecture"
  content="Migraremos a JWT + Refresh Tokens rotativos en lugar de sesiones en base de datos"
  reason="Necesitamos soportar escalado horizontal y clientes móviles/offline. Las sesiones en BD generan problemas de consistencia y dificultan el scaling."
  alternatives=["Sesiones en Redis", "OAuth2 + DB sessions", "Magic links sin tokens"]
  tags=["auth", "architecture", "scalability", "mobile"]
```

Semanas después, cuando alguien pregunta por qué no usamos sesiones como antes, esta memoria aparece automáticamente vía `get-context`.

### Ejemplo 2: Fix no obvio de concurrencia

Encuentras un memory leak sutil:

```text
save-decision
  key="worker-pool-leak-fix"
  decisionType="fix"
  content="El memory leak en el worker pool se producía porque las goroutines internas no respetaban la cancelación del contexto"
  reason="El bug solo aparecía bajo alta carga + cancelaciones frecuentes. El context se cancelaba pero los workers internos seguían corriendo."
  alternatives=["Aumentar límites de memoria", "Usar sync.Pool", "Workers efímeros por petición"]
  tags=["bug", "concurrency", "performance"]
```

Este tipo de fix es muy valioso porque el mismo patrón puede repetirse en otros servicios.

### Ejemplo 3: Convención de equipo

Estableces un estándar de manejo de errores:

```text
save-decision
  key="error-handling-convention"
  decisionType="convention"
  content="Todos los errores de dominio deben implementar DomainError con método Code() y Unwrap()"
  reason="Queremos códigos de error consistentes para el frontend y poder hacer wrapping sin perder la causa original."
  tags=["convention", "errors", "api"]
```

Este tipo de decisión afecta a todo el código nuevo que se escriba y es perfecta para promover.

## Cuándo NO promover (ejemplos concretos)

- “Probé cambiar X y no funcionó” → Contexto de sesión.
- “Recuerda revisar el PR #234 mañana” → Nota temporal.
- “Usamos Gin porque ya lo sabíamos” → Decisión débil, sin alternatives reales.
- Cualquier cosa que ya esté bien documentada en `ARCHITECTURE.md` o en el código.

## Criterios para promoción

Promueve cuando el conocimiento:

- Probablemente lo necesitarás recordar dentro de meses.
- Es difícil de deducir solo mirando el código.
- Afecta a decisiones futuras del equipo.
- Tiene valor más allá de la sesión actual.