# Backlog tecnico priorizado

Este backlog traduce [plan2.md](../plan2.md) a issues concretas, en orden recomendado de ejecucion.

## Prioridad inmediata

1. [TECH-01-cold-start-get-context.md](./TECH-01-cold-start-get-context.md)
2. [TECH-02-context7-library-id-flow.md](./TECH-02-context7-library-id-flow.md)
3. [TECH-03-monorepo-dependency-analysis.md](./TECH-03-monorepo-dependency-analysis.md)

## Prioridad siguiente

4. [TECH-04-memory-limits-eviction.md](./TECH-04-memory-limits-eviction.md)
5. [TECH-05-src-context-mcp-enrichment.md](./TECH-05-src-context-mcp-enrichment.md)

## Prioridad posterior

6. [TECH-06-session-memory-promotion.md](./TECH-06-session-memory-promotion.md)

## Dependencias entre issues

- `TECH-01` desbloquea integracion real con `src/context`.
- `TECH-02` desbloquea una integracion correcta con documentacion viva.
- `TECH-03` mejora la calidad de contexto en workspaces mixtas.
- `TECH-04` estabiliza la memoria antes de integrarla mas.
- `TECH-05` debe comenzar despues de `TECH-01` y preferiblemente despues de `TECH-02`.
- `TECH-06` debe comenzar cuando `TECH-04` ya este estable.

## Nota de alcance

Este backlog evita dos errores:

1. crecer en numero de tools antes de corregir precision
2. intentar replicar `MemPalace` como sistema generalista

La direccion correcta es fortalecer `mcp-go-context` como capa Go de contexto tecnico y memoria operativa.