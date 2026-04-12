# Issue TEC-01: hacer que get-context funcione bien en arranque en frio

**Etiquetas**: `bug`, `core`, `context`, `priority-high`, `mcp`

## Problema

`get-context` depende en la practica de que exista cache previa del analizador. Si no se ha ejecutado antes `analyze-project`, el contexto relevante puede ser pobre o vacio.

## Impacto

- mala primera experiencia de uso
- integracion fragil con clientes MCP
- bloquea consumo directo desde `src/context`

## Objetivo

Hacer que `get-context` sea autosuficiente y util sin warm-up manual.

## Alcance

### Cambios esperados

1. Si la cache del analizador esta vacia, ejecutar un discovery minimo del proyecto.
2. Separar internamente indexado rapido y extraccion de contexto.
3. Si la query ya incluye archivos concretos, evitar recorrido completo innecesario.
4. Mejorar ranking con señales minimas:

- nombre de archivo
- extension o lenguaje
- recencia
- contenido basico cuando el coste sea bajo
- cambios Git cuando exista repositorio

5. Mantener truncado estable por presupuesto de salida.

### Fuera de alcance

- analisis semantico profundo por simbolos
- embeddings
- cache incremental avanzada

## Archivos probables

- `internal/analyzer/analyzer.go`
- `internal/tools/tools.go`

## Criterios de aceptacion

- `get-context` devuelve contexto util sin `analyze-project` previo
- la latencia sigue siendo razonable en repos medianos
- hay tests para arranque en frio y arranque en caliente
- `go test ./...` pasa

## Definicion de terminado

Un cliente MCP puede invocar `get-context` directamente y obtener una respuesta util en la primera llamada.