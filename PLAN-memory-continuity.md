# Plan: continuidad de memoria entre aplicaciones

Estado: implementado en el repositorio. La verificación automática cubre almacenamiento compartido, identidad, conflictos y conservación de recuerdos promovidos. La prueba manual Claude Desktop ↔ OpenCode queda pendiente de reiniciar los clientes con el binario nuevo.

## 1. Objetivo

Permitir que Claude, OpenCode y otros clientes MCP puedan continuar el trabajo de un proyecto compartiendo:

- Decisiones duraderas.
- Estado confirmado del trabajo.
- Comprobaciones realizadas.
- Tareas pendientes.
- Siguiente paso recomendado.

**Escenario de aceptación principal:**

> Claude guarda el estado de un trabajo. OpenCode, que ya estaba abierto, consulta el mismo proyecto y recupera ese estado sin reiniciar el servidor. Ambos pueden guardar cambios sin perder información.

La continuidad dependerá de que el asistente invoque las herramientas. Las instrucciones de cada cliente establecerán cuándo hacerlo.

## 2. Alcance de la primera versión

Incluye:

- Resolución estable del proyecto desde una ruta.
- Almacenamiento compartido fiable entre procesos locales.
- Herramientas `resume-context` y `save-handoff`.
- Protección efectiva de recuerdos promovidos.
- Instrucciones de uso para OpenCode, Claude Code y Claude Desktop.
- Compatibilidad y migración de recuerdos existentes.

Quedan para una fase posterior:

- Sincronización entre equipos.
- Embeddings o búsqueda semántica.
- Captura automática de conversaciones.
- Plugins específicos de inicio de cada aplicación.
- Rediseño del dashboard.

La primera versión debe funcionar con varios clientes en **el mismo equipo y usando el mismo directorio de almacenamiento**.

## 3. Diseño funcional

### 3.1. Identidad del proyecto

Crear un componente de resolución de proyectos en `internal/project/`.

**Resolución:**

1. Recibir la ruta de trabajo explícita.
2. Normalizarla y localizar la raíz Git si existe.
3. Consultar su asociación con un `projectId` en un registro local compartido.
4. Al guardar por primera vez, crear una identidad si no existe.
5. Permitir asociar otra ruta a una identidad existente de forma explícita.

El nombre visible será descriptivo; la identidad interna no dependerá únicamente del nombre de la carpeta.

**Reglas:**

- Abrir una subcarpeta debe resolver el mismo proyecto.
- Dos repositorios con el mismo nombre deben permanecer separados.
- Una consulta no creará archivos dentro del repositorio.
- Git remoto puede ayudar a reconocer una copia, pero no decidirá automáticamente que dos repositorios son el mismo.
- Los worktrees compartirán identidad de proyecto y conservarán su contexto de trabajo diferenciado.
- Fuera de Git, la ruta explícita se tratará como raíz; se podrán registrar alias.

El proyecto se resolverá **por operación**. No se cambiará un “proyecto activo” global que pueda afectar a otro cliente conectado.

### 3.2. Dos tipos de información

| Tipo | Contenido | Conservación |
|---|---|---|
| Memoria duradera | Decisiones, convenciones, soluciones reutilizables | Promoción y conservación explícita |
| Resumen de continuidad —handoff— | Objetivo, progreso, verificaciones, pendientes y próximo paso | Actualizable, fechado y con historial limitado |

Los resúmenes no se promoverán automáticamente.

**Campos del resumen:**

```text
schemaVersion
projectId
handoffId
revision
updatedAt
sourceClient
worktree / branch / commit, cuando estén disponibles
objective
completed[]
verification[]
pending[]
nextStep
references[]
```

El servidor calculará fecha y datos Git cuando pueda verificarlos. La procedencia declarada por el asistente se distinguirá de los datos observados por el servidor.

Habrá resúmenes separados por contexto de trabajo para evitar que dos ramas o tareas independientes se sobrescriban.

### 3.3. Herramientas MCP

#### `resume-context`

Entradas:

- `path`: ruta de trabajo.
- `projectId`: opcional, para identificación explícita.
- `query`: objetivo de la sesión, opcional.
- `maxTokens`: presupuesto aproximado.
- `handoffId`: opcional, para retomar un trabajo concreto.

Respuesta:

- Identidad del proyecto.
- Resumen correspondiente al contexto actual.
- Decisiones relevantes.
- Fecha, procedencia y revisión.
- Diferencias de rama o commit.
- Referencias para ampliar la información.
- Indicación clara de ausencia o ambigüedad de contexto.

Si existen varios trabajos posibles, devolverá candidatos breves. No elegirá silenciosamente el resumen más reciente de otra rama.

El presupuesto se aplicará a toda la respuesta con una estimación coherente de tokens.

#### `save-handoff`

Entradas:

- `path` o identidad explícita.
- Campos del resumen.
- `handoffId`, cuando se actualice uno existente.
- `expectedRevision`, para comprobar que no se está sobrescribiendo una versión posterior.

Comportamiento:

- Resolver el proyecto.
- Validar tamaños y campos.
- Guardar de forma transaccional.
- Devolver identificador y nueva revisión.
- Ante conflicto, conservar ambos trabajos y pedir reconciliación mediante un resultado reconocible.

No fusionará automáticamente afirmaciones contradictorias.

### 3.4. Instrucciones de los clientes

Preparar bloques pequeños para:

- **OpenCode:** `AGENTS.md`.
- **Claude Code:** `CLAUDE.md`.
- **Claude Desktop:** instrucciones de proyecto.

Flujo recomendado:

1. Al iniciar trabajo sustantivo, consultar `resume-context`.
2. Contrastar el resumen con archivos y Git.
3. Guardar decisiones importantes mediante las herramientas de memoria.
4. Actualizar el resumen tras un hito, antes de cambiar de aplicación o al cerrar el trabajo.

No depender exclusivamente del último mensaje: una sesión puede terminar inesperadamente.

Los archivos de instrucciones contendrán el procedimiento; el estado cambiante estará en el MCP.

## 4. Fases de implementación

### Fase 1 — Base de pruebas y almacenamiento fiable

**Archivos principales:** `internal/memory/manager.go`, pruebas actuales y nuevos archivos de almacenamiento.

- [ ] Corregir la colisión de `contains` que impide compilar las pruebas de memoria.
- [ ] Separar acceso al almacenamiento y lógica de selección de recuerdos.
- [ ] Mantener JSON como formato inicial.
- [ ] Implementar bloqueo entre procesos con soporte Windows y Unix.
- [ ] Aplicar a cada escritura el ciclo: bloquear → leer versión actual → modificar → persistir atómicamente.
- [ ] Refrescar las lecturas desde el almacenamiento compartido.
- [ ] Incluir promoción, borrado, limpieza y estadísticas en la misma disciplina de acceso.
- [ ] Propagar errores de persistencia sin confirmar operaciones fallidas.

La selección concreta del mecanismo de bloqueo deberá comprobar liberación tras un cierre inesperado y sustitución atómica en Windows. Un mutex de Go o un simple archivo temporal no bastan.

**Terminado cuando:** dos procesos independientes conservan todas las escrituras y ven las actualizaciones del otro sin reiniciarse.

### Fase 2 — Identidad y migración compatibles

**Archivos principales:** nuevo `internal/project/`, `internal/config/`, `internal/memory/` y `internal/server/`.

- [ ] Implementar resolución de raíz y registro de identidades.
- [ ] Asociar recuerdos por `(projectId, key)`.
- [ ] Incorporar ámbito explícito en guardado y recuperación.
- [ ] Mantener las llamadas antiguas mediante un comportamiento de compatibilidad definido.
- [ ] Unificar el filtrado de proyecto en decisiones, promociones y búsquedas.
- [ ] Versionar el almacenamiento.
- [ ] Crear copia de respaldo antes de migrar.
- [ ] Preservar recuerdos heredados sin asignación segura; no adjudicarlos por coincidencia de nombre.
- [ ] Hacer la migración repetible sin duplicar datos.

**Condición de despliegue:** detener los servidores antiguos antes de migrar. El binario anterior no conoce los nuevos bloqueos y no debe seguir escribiendo simultáneamente.

**Terminado cuando:** las memorias existentes siguen accesibles y dos proyectos con claves iguales no interfieren entre sí.

### Fase 3 — Continuidad entre clientes

**Archivos principales:** nuevos archivos de handoffs, `internal/tools/` e `internal/server/server.go`.

- [ ] Implementar almacenamiento de resúmenes con revisiones.
- [ ] Registrar `save-handoff` y `resume-context`.
- [ ] Detectar actualizaciones concurrentes.
- [ ] Seleccionar por proyecto y contexto Git.
- [ ] Mostrar antigüedad, procedencia y desajustes.
- [ ] Aplicar límites de entrada, respuesta e historial.
- [ ] Adaptar interfaces y dobles de prueba existentes.

**Terminado cuando:** Claude puede dejar un resumen que OpenCode recupera, con identificación y siguiente paso correctos.

### Fase 4 — Memoria duradera que apoye la continuidad

**Archivos principales:** `internal/memory/manager.go` y `internal/tools/tools.go`.

- [ ] Proteger recuerdos promovidos frente a caducidad y expulsión automática.
- [ ] Preservar promoción y metadatos al actualizar una memoria.
- [ ] Evitar devolver recuerdos ajenos a una consulta solo por ser recientes.
- [ ] Incluir clave, motivo y alternativas en la búsqueda.
- [ ] Dar preferencia a recuerdos promovidos cuando sean relevantes.
- [ ] Corregir el presupuesto de contexto de `get-context`.

Si la capacidad se agota con recuerdos protegidos, devolver un resultado explícito en lugar de eliminarlos silenciosamente.

**Terminado cuando:** una decisión promovida sobrevive a la limpieza y las consultas sin coincidencias no reciben ruido.

### Fase 5 — Instrucciones, documentación y validación real

**Archivos principales:** instrucciones de inicialización, `AGENTS.md`, `README.md` y guía de configuración en `website/`.

- [ ] Documentar el flujo independiente del cliente.
- [ ] Incluir bloques listos para copiar en cada aplicación.
- [ ] Documentar que el servidor y los clientes pueden tener directorios de trabajo distintos: la ruta explícita es preferible.
- [ ] Explicar configuración compartida, migración y recuperación del respaldo.
- [ ] Actualizar las instrucciones de este repositorio.
- [ ] Probar el flujo real Claude → OpenCode y OpenCode → Claude.
- [ ] Registrar resultados en el plan.

Las configuraciones globales de aplicaciones se tratarán como un paso de despliegue explícito, separado de los cambios del repositorio.

## 5. Verificación

| Prueba | Resultado esperado |
|---|---|
| Dos procesos guardan recuerdos distintos | Ambos recuerdos permanecen |
| Un proceso ya abierto consulta después de una escritura externa | Ve el dato nuevo |
| Dos clientes actualizan la misma revisión | Uno recibe conflicto; no hay sobrescritura silenciosa |
| Abrir la raíz o una subcarpeta | Mismo proyecto |
| Dos repositorios con igual nombre | Identidades distintas |
| Dos ramas o worktrees con trabajos diferentes | Resúmenes distinguibles |
| Recuerdo promovido antiguo y limpieza activada | Se conserva |
| Consulta sin coincidencias | Sin recuerdos irrelevantes |
| Resumen de otra revisión Git | Diferencia visible |
| Migración repetida | Sin pérdidas ni duplicados |
| Interrupción durante persistencia | Permanece un estado válido recuperable |

Comandos de validación (los paquetes nuevos se comprobarán después de crearlos):

```text
go test ./internal/memory ./internal/tools ./internal/config
go test ./internal/project ./internal/server
go test ./...
go vet ./...
go build ./cmd/mcp-context-server
```

Añadir pruebas multiproceso con almacenamiento temporal y ejecutar `-race` donde el entorno lo permita. Las pruebas no utilizarán la memoria personal real.

## 6. Entrega y criterio final de éxito

La entrega incluirá:

1. Plan actualizado con resultados.
2. Implementación y pruebas.
3. Migración documentada y respaldo.
4. Instrucciones para los tres clientes.
5. Demostración de continuidad bidireccional.

**El trabajo estará completo cuando puedas cambiar de aplicación, recuperar el contexto correcto y continuar sin volver a explicar el proyecto ni perder lo guardado por el otro cliente.**

**Orden recomendado:** fiabilidad → identidad → continuidad → recuperación → integración.

## 7. Resultado de la implementación

Implementado:

- Bloqueo entre procesos y escritura atómica.
- Recarga desde disco en cada operación de memoria.
- Identidad de proyecto por ruta, con raíces Git y nombres de carpeta duplicados separados.
- `resume-context` y `save-handoff`, con conflicto que conserva ambas versiones.
- Recuerdos promovidos protegidos frente a caducidad y expulsión.
- Actualizaciones que conservan promoción, uso y metadatos.
- Búsqueda que no devuelve recuerdos sin coincidencia.
- Instrucciones en `AGENTS.md`, `CLAUDE.md`, `README.md` y la guía `website/src/content/docs/guides/continuity.md`.

Pendiente operativo: recompilar el binario, detener servidores antiguos y comprobar manualmente Claude Desktop ↔ OpenCode con el mismo directorio de memoria.
