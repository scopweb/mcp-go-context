# Plan de ejecucion detallado de mcp-go-context

## Objetivo operativo

Convertir `mcp-go-context` en la capa principal de contexto tecnico y memoria operativa que pueda ser consumida por `src/context`, usando `Context7` como proveedor especializado de documentacion actualizada y tomando de `MemPalace` solo ideas de producto, no dependencias.

Este plan asume tres restricciones no negociables:

- implementacion 100% Go dentro de `mcp-go-context`
- prioridad a integracion real sobre expansion cosmetica de features
- precision y robustez antes que numero de tools

## Diagnostico que justifica este plan

### Problemas principales detectados

1. `get-context` no es fiable en arranque en frio.
2. La integracion actual con `Context7` no sigue el flujo oficial por `libraryId`.
3. `src/context` no consume hoy el valor real de `mcp-go-context`.
4. El analisis multi-ecosistema existe, pero es todavia superficial para monorepos.
5. Hay duplicacion entre memoria local de sesion y memoria MCP persistente.

### Decision de producto

`mcp-go-context` no debe intentar replicar `MemPalace` como sistema generalista de memoria semantica.

Debe posicionarse como:

- servidor MCP de contexto local
- capa de analisis tecnico para repositorios reales
- memoria operativa de decisiones, fixes y continuidad de trabajo
- backend de enriquecimiento para `src/context`

## Resultado final deseado

Al terminar este plan, el flujo ideal debe ser este:

1. `src/context` detecta que hay disponible un servidor `mcp-go-context`.
2. En primer uso, solicita un analisis basico del repo sin requerir precalentamiento manual.
3. Para preguntas de codigo, solicita contexto relevante al servidor.
4. Para dependencias o APIs, el servidor consulta `Context7` con el flujo correcto.
5. Para decisiones y continuidad, el servidor persiste memoria operativa reutilizable.
6. La experiencia sigue siendo simple: un binario Go, una config, cero runtimes extra.

## Principios de ejecucion

1. Primero corregir comportamiento incorrecto.
2. Luego reforzar precision.
3. Despues integrar con `src/context`.
4. Solo al final ampliar producto y UX.

## Fase 0: endurecimiento de la base existente

Objetivo: asegurar que lo que ya existe funciona de forma coherente y predecible.

### Tareas

1. Revisar todas las tools registradas y validar que cada una tiene comportamiento util en arranque en frio.
2. Confirmar consistencia entre config por defecto, config cargada desde archivo y uso real en runtime.
3. Revisar memoria para asegurar aplicacion real de `MaxEntries`, `MaxResults`, TTL y eviction.
4. Revisar si `StoragePath` se trata siempre como directorio y nunca como archivo ambiguo.
5. Añadir pruebas de integracion minimas por tool critica.

### Archivos objetivo

- `internal/config/config.go`
- `internal/memory/manager.go`
- `internal/server/server.go`
- `internal/tools/tools.go`

### Criterios de aceptacion

- ninguna tool critica depende de estado previo no documentado
- configuracion por defecto coherente con el comportamiento real
- memoria limitada por politicas reales y no solo por campos de config
- `go test ./...` pasa en local

## Fase 1: corregir el nucleo de contexto

Objetivo: hacer que `get-context` sea util por si solo y deje de depender de llamadas previas a `analyze-project`.

### Problema concreto

Hoy el ranking de archivos relevantes depende de la cache del analizador. Eso significa que, si no hubo analisis previo, `get-context` puede quedarse sin material.

### Tareas

1. Cambiar el flujo de `GetRelevantContext` para que haga discovery minimo si la cache esta vacia.
2. Separar dos pasos internos.

Paso A: indexado rapido del proyecto.

Paso B: ranking y extraccion de contexto.

3. Permitir un modo lightweight que no recorra todo el repo si la consulta ya viene con archivos concretos.
4. Mejorar el ranking mezclando al menos estas señales:

- nombre de archivo
- extensiones y tipo de archivo
- contenido basico cuando sea barato
- recencia
- cambios Git cuando existan

5. Añadir truncado por presupuesto de tokens o caracteres con salida estable.

### Archivos objetivo

- `internal/analyzer/analyzer.go`
- `internal/tools/tools.go`

### Criterios de aceptacion

- `get-context` devuelve contexto util sin necesitar `analyze-project` previo
- el tiempo de respuesta sigue siendo razonable en repos medianos
- las pruebas cubren arranque en frio y arranque en caliente

### Entregable

Un `get-context` que pueda ser invocado directamente desde `src/context` o desde cualquier cliente MCP sin dependencia de warm-up manual.

## Fase 2: integrar Context7 correctamente

Objetivo: dejar de usar una llamada ad hoc y adoptar el flujo real de `Context7`.

### Problema concreto

La integracion actual consulta una API formada a mano por nombre de libreria. `Context7` hoy trabaja por resolucion de `libraryId` y luego consulta de docs por ese identificador.

### Tareas

1. Reemplazar `fetchFromContext7` por un flujo de dos pasos.

Paso 1: resolver `libraryId` usando nombre de libreria mas query.

Paso 2: recuperar contexto con `libraryId` exacto.

2. Rediseñar la entrada de `fetch-docs` para aceptar mejor el caso real.

Campos recomendados:

- `library`
- `query`
- `version`
- `libraryId` opcional

3. Si el usuario ya aporta `libraryId`, saltar la resolucion.
4. Si la resolucion falla, usar fallback local de documentacion solo como degradacion controlada.
5. Registrar errores y respuestas no encontradas de forma clara.

### Archivos objetivo

- `internal/tools/tools.go`
- posiblemente `internal/server/server.go`

### Criterios de aceptacion

- `fetch-docs` usa `Context7` con contrato compatible con su API actual
- el fallback local solo entra cuando la resolucion externa falla o no hay conectividad
- la salida final indica claramente si la respuesta vino de `Context7` o de fallback local

### Entregable

Una tool `fetch-docs` robusta y alineada con el proveedor real, apta para ser usada por `src/context` y por agentes MCP.

## Fase 3: mejorar el analisis multi-ecosistema sin salir de Go

Objetivo: hacer que el servidor sea util en workspaces mixtas como esta.

### Problema concreto

Ya existe soporte para `go.mod`, `package.json`, `pyproject.toml` y `requirements.txt`, pero sigue siendo demasiado plano y centrado en un solo `ProjectPaths[0]`.

### Tareas

1. Cambiar el analisis de dependencias para iterar sobre todos los `ProjectPaths` configurados.
2. Añadir deteccion de manifests por recorrido controlado dentro de cada raiz.
3. Soportar monorepos con varios `package.json` y varios `go.mod`.
4. Distinguir dependencias por scope.

Scopes minimos:

- root
- package
- module
- app o service

5. Mejorar el parser de `pyproject.toml` sin introducir librerias pesadas si no son necesarias.
6. Incorporar lectura de `pnpm-workspace.yaml` y estructuras similares si el coste sigue siendo bajo.

### Archivos objetivo

- `internal/analyzer/analyzer.go`
- `internal/config/config.go`

### Criterios de aceptacion

- el analisis refleja multiples subproyectos reales
- la salida no mezcla dependencias de forma ambigua
- las pruebas incluyen casos con monorepo mixto

### Entregable

Un `dependency-analysis` realmente util fuera del caso Go puro.

## Fase 4: memoria operativa con foco de desarrollo

Objetivo: consolidar una memoria util para continuidad de trabajo, sin perseguir un sistema universal.

### Direccion elegida

Priorizar memorias de alto valor para desarrollo:

- decisiones tecnicas
- fixes y postmortems
- contexto de debugging
- onboarding de repo
- convenciones del proyecto

### Tareas

1. Estabilizar `search-memory`, `save-decision` y `get-decisions` como trio base.
2. Mejorar ranking con mezcla de texto, tags, recencia y uso.
3. Añadir esquema minimo de tipos de decision y tags sugeridos.
4. Añadir tool opcional `save-fix` si la diferencia con `save-decision` se justifica.
5. Añadir export simple a Markdown o JSON para inspeccion local.

### No hacer en esta fase

- embeddings
- base vectorial externa
- compresion tipo AAAK
- taxonomia compleja tipo palace

### Criterios de aceptacion

- decisiones recuperables por tipo y keyword
- resultados ordenados de forma util para sesiones reales
- persistencia simple y legible en disco

## Fase 5: integracion con src/context

Objetivo: que `src/context` deje de ser solo contexto local y pueda enriquecerse desde `mcp-go-context`.

### Estrategia

No mover toda la logica de `src/context` a MCP. Mantener su rol de contexto base rapido y local, y añadir una capa de enriquecimiento opcional basada en disponibilidad de cliente MCP.

### Propuesta de integracion

1. Mantener en `src/context` lo actual.

- git status
- fecha
- CLAUDE.md y archivos de memoria locales

2. Añadir enriquecimiento opcional desde MCP para consultas relevantes.

Casos iniciales:

- resumen estructural del proyecto
- archivos cambiados recientemente
- contexto tecnico por query
- documentacion de dependencias

3. No bloquear el arranque si el MCP no esta disponible.
4. Cachear respuestas breves por sesion cuando tenga sentido.
5. Diseñar degradacion limpia.

Si no hay MCP conectado, el sistema sigue operando con el contexto actual.

### Orden recomendado

1. Integrar `analyze-project` o una variante resumida para inicio de sesion.
2. Integrar `changed-files-context` para ramas activas con cambios.
3. Integrar `get-context` bajo demanda para preguntas tecnicas.
4. Integrar `fetch-docs` para dependencias.

### Criterios de aceptacion

- `src/context` mejora contexto sin romper tiempos base de respuesta
- la integracion es opcional y tolerante a fallos
- no se duplica innecesariamente el trabajo ya hecho por `src/context`

## Fase 6: convergencia con SessionMemory

Objetivo: reducir la duplicacion entre memoria de sesion local y memoria persistente MCP.

### Problema concreto

Hoy `SessionMemory` y `mcp-go-context` viven en paralelo y no se alimentan entre si.

### Tareas

1. Definir que informacion debe seguir solo en `summary.md` local.
2. Definir que informacion debe promocionarse a memoria persistente.

Promocion inicial recomendada:

- decisiones tecnicas confirmadas
- fixes relevantes
- convenciones del repo

3. Diseñar una interfaz minima de promocion, no una sincronizacion total.
4. Evitar guardar ruido transitorio.
5. Mantener la extraccion local barata y delegar persistencia solo a items de alto valor.

### Criterios de aceptacion

- no hay doble escritura indiscriminada
- la memoria persistente contiene informacion mas estable y accionable
- la memoria de sesion sigue sirviendo para compaction local

## Fase 7: producto, DX y documentacion

Objetivo: cuando la base funcione, hacer que se entienda y se adopte mejor.

### Tareas

1. Actualizar README con workflows reales.
2. Documentar integracion con `src/context` como caso de referencia.
3. Añadir ejemplos de uso por escenario.

Escenarios recomendados:

- onboarding de repo
- recuperar decisiones pasadas
- obtener contexto de cambios recientes
- traer docs actuales de una dependencia

4. Refinar mensajes de error y salidas de tools.
5. Añadir benchmarks simples de latencia y tamaño de contexto.

### Criterios de aceptacion

- el README explica claramente para que sirve y para que no sirve
- los ejemplos muestran valor real en repos mixtos
- la experiencia de primera ejecucion es clara

## Orden de implementacion recomendado

### Prioridad inmediata

1. Fase 1
2. Fase 2
3. Fase 3
4. Fase 5

### Prioridad siguiente

1. Fase 4
2. Fase 6

### Prioridad final

1. Fase 7

## Backlog concreto de issues

### Bloque A: fiabilidad del nucleo

- `get-context` debe hacer indexado minimo cuando la cache esta vacia
- `get-context` debe soportar mejor presupuesto de tokens
- `analyze-project` y `get-context` deben compartir pipeline interno coherente

### Bloque B: Context7

- reemplazar API v1 ad hoc por resolucion de `libraryId`
- rediseñar schema de `fetch-docs`
- marcar fuente de respuesta: remoto o local

### Bloque C: monorepo y multi-ecosistema

- iterar `ProjectPaths` completos
- detectar manifests multiples por workspace
- mejorar salida de `dependency-analysis`

### Bloque D: memoria operativa

- endurecer limites y eviction
- estabilizar `search-memory`
- estabilizar `save-decision`
- valorar `save-fix`

### Bloque E: integracion externa

- diseñar enriquecimiento MCP en `src/context`
- definir politica de degradacion si no hay servidor conectado
- definir promocion minima desde SessionMemory a memoria persistente

## Riesgos a evitar

1. Intentar copiar `MemPalace` en vez de aprender de sus aciertos de producto.
2. Añadir mas tools antes de corregir la precision de las actuales.
3. Meter dependencias Go pesadas para parsing si un parser simple resuelve el 80% del caso.
4. Acoplar `src/context` de forma dura a un servidor MCP no garantizado.
5. Convertir la memoria en un vertedero de texto sin promocion selectiva.

## Definicion de exito

Se considerara que el plan tuvo exito cuando se cumplan estas condiciones:

1. `mcp-go-context` puede responder con contexto util desde cero, sin precalentamiento manual.
2. `fetch-docs` usa `Context7` de forma correcta y trazable.
3. El servidor entiende repositorios Go, TypeScript y Python de forma suficiente para desarrollo cotidiano.
4. `src/context` puede aprovechar `mcp-go-context` sin perder resiliencia ni simplicidad.
5. La memoria persistente recoge decisiones y continuidad reales, no ruido.

## Siguiente sprint recomendado

### Sprint 1

1. Corregir `get-context` para arranque en frio.
2. Añadir tests de regresion para ese flujo.
3. Rediseñar `fetch-docs` con `libraryId`.

### Sprint 2

1. Expandir `dependency-analysis` a monorepo real.
2. Mejorar ranking con senales de Git.
3. Probar integracion minima con `src/context`.

### Sprint 3

1. Consolidar memoria operativa.
2. Definir promocion desde SessionMemory.
3. Actualizar documentacion y ejemplos.

## Nota final de enfoque

La ventaja competitiva de `mcp-go-context` no sera parecerse a todo.

Sera resolver bien un problema muy concreto:

> dar a un asistente de codigo contexto tecnico local, documentacion actualizada y memoria operativa suficiente, con una implementacion pequeña, fiable y 100% Go.