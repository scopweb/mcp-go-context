# Issue TEC-06: definir promocion selectiva desde SessionMemory a memoria persistente

**Etiquetas**: `integration`, `memory`, `session`, `priority-low`, `design`

## Problema

La memoria de sesion local y la memoria persistente MCP viven hoy separadas. Eso crea duplicacion conceptual y dificulta recuperar decisiones duraderas.

## Impacto

- split-brain entre memoria local y memoria persistente
- decisiones importantes pueden quedarse solo en `summary.md`
- la memoria persistente puede llenarse de ruido si se integra mal

## Objetivo

Definir una promocion minima y selectiva desde SessionMemory hacia memoria persistente, sin sincronizacion total.

## Alcance

### Cambios esperados

1. Definir que contenido se queda solo en memoria de sesion.
2. Definir que contenido se promociona.

Promocion inicial recomendada:

- decisiones tecnicas confirmadas
- fixes relevantes
- convenciones del repo

3. Diseñar una interfaz minima de promocion.
4. Evitar doble escritura indiscriminada.
5. Mantener SessionMemory ligera para compaction local.

### Dependencias

- `TECH-04`
- idealmente despues de estabilizar `save-decision` y `search-memory`

### Fuera de alcance

- sincronizacion bidireccional completa
- persistencia de todo el transcript

## Criterios de aceptacion

- la memoria persistente contiene informacion estable y accionable
- la memoria de sesion sigue siendo barata y local
- no hay promocion automatica de ruido transitorio

## Definicion de terminado

Existe una politica clara y minimalista para subir solo el contenido de alto valor a memoria persistente.

## Estado actual (2026)

**Avance significativo implementado**:
- Se creó la tool `suggest-promotions` como interfaz principal de descubrimiento selectivo.
- Se implementó `SuggestForPromotion()` en el memory manager con una heurística transparente y mejorada (estructura de decisión, usage, recencia, keywords ponderadas, tipo de decisión, alternativas y tags).
- El flujo recomendado ahora es: `save-decision` → `suggest-promotions` → `promote-memory`.
- Se mejoraron las instrucciones del `initialize` y las descripciones de tools para guiar mejor a los modelos.
- Tests dedicados para la lógica de sugerencias.

Esto cumple en gran medida con el objetivo de "interfaz minima de promocion" sin sincronización completa ni ruido automático. La promoción sigue siendo explícita y de alta calidad.