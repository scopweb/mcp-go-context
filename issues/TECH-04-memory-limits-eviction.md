# Issue TEC-04: endurecer limites, TTL y eviction de memoria

**Etiquetas**: `bug`, `memory`, `priority-medium`, `stability`

## Problema

La memoria persistente expone configuracion de limites, TTL y numero maximo de entradas, pero hay que asegurar que esas politicas se aplican de forma real y consistente.

## Impacto

- crecimiento no controlado de memoria
- resultados poco predecibles
- base debil para integrar memoria con `src/context` y SessionMemory

## Objetivo

Hacer que la memoria operativa tenga limites reales, ordenacion estable y limpieza basica confiable.

## Alcance

### Cambios esperados

1. Aplicar de verdad `MaxEntries`.
2. Aplicar de verdad `MaxResults`.
3. Implementar limpieza por TTL a nivel de sesiones o memorias si corresponde.
4. Implementar eviction simple por recencia y uso.
5. Revisar consistencia de `StoragePath` como directorio.

### Fuera de alcance

- memoria vectorial
- compresion avanzada
- indexacion semantica externa

## Archivos probables

- `internal/memory/manager.go`
- `internal/config/config.go`

## Criterios de aceptacion

- las politicas de limite se respetan
- la memoria no crece indefinidamente
- el orden de resultados es estable y justificable
- hay tests de eviction y TTL

## Definicion de terminado

La memoria persistente es lo bastante robusta como para ser consumida por otras capas sin comportamientos sorpresa.