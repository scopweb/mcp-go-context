# Issue TEC-05: diseñar integracion de enriquecimiento desde src/context

**Etiquetas**: `integration`, `src-context`, `priority-medium`, `design`, `mcp`

## Problema

`src/context` hoy solo construye contexto local base y no aprovecha `mcp-go-context`, a pesar de que la infraestructura MCP ya existe.

## Impacto

- duplicacion de responsabilidades
- contexto tecnico menos rico del que podria ser
- nula reutilizacion del servidor Go desde la app principal

## Objetivo

Diseñar e implementar una integracion opcional y tolerante a fallos desde `src/context` hacia `mcp-go-context`.

## Alcance

### Cambios esperados

1. Mantener el contexto local actual como base.
2. Añadir enriquecimiento MCP no bloqueante.
3. Integrar progresivamente estas capacidades:

- analisis estructural basico del repo
- changed files context
- get-context por query
- fetch-docs para dependencias

4. Añadir politica de degradacion limpia si el cliente MCP no esta disponible.
5. Añadir cache breve por sesion si el coste de llamadas lo justifica.

### Dependencias

- `TECH-01`
- recomendable despues de `TECH-02`

### Fuera de alcance

- mover toda la logica de `src/context` al servidor MCP
- hacer obligatorio el uso de `mcp-go-context`

## Criterios de aceptacion

- la app sigue funcionando si no hay MCP conectado
- el enriquecimiento mejora el contexto sin romper tiempos base
- no se duplica trabajo innecesariamente

## Definicion de terminado

`src/context` puede aprovechar valor real de `mcp-go-context` sin acoplarse de forma dura a su disponibilidad.