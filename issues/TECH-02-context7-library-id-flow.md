# Issue TEC-02: alinear fetch-docs con el flujo real de Context7

**Etiquetas**: `bug`, `integration`, `context7`, `priority-high`, `docs`

## Problema

La integracion actual con `Context7` construye una llamada ad hoc por nombre de libreria y version. El proveedor actual trabaja por resolucion de `libraryId` y luego consulta del contexto por ese identificador.

## Impacto

- menor precision en desambiguacion
- riesgo de incompatibilidad con el contrato actual del proveedor
- peor soporte para versiones y librerias con nombres ambiguos

## Objetivo

Reemplazar la integracion manual por un flujo compatible con la API actual de `Context7`.

## Alcance

### Cambios esperados

1. Rediseñar `fetchFromContext7` a flujo de dos pasos.

Paso 1: resolver `libraryId`.

Paso 2: consultar docs o contexto por `libraryId`.

2. Ajustar el input schema de `fetch-docs`.

Campos recomendados:

- `library`
- `query`
- `version`
- `libraryId` opcional

3. Si se recibe `libraryId`, saltar resolucion.
4. Si la resolucion falla, usar fallback local de forma explicita.
5. Informar claramente si la salida viene de `Context7` o de fallback local.

### Fuera de alcance

- montar un mirror local de `Context7`
- reimplementar ranking del proveedor

## Archivos probables

- `internal/tools/tools.go`
- `internal/server/server.go`

## Criterios de aceptacion

- `fetch-docs` usa un flujo compatible con la API actual
- hay soporte para query por `libraryId`
- el fallback local queda como degradacion controlada
- los errores son legibles y trazables

## Definicion de terminado

`fetch-docs` puede ser usado desde clientes MCP modernos sin depender de una API heredada formada manualmente.