# Plan de expansion de mcp-go-context

## Objetivo

Expandir `mcp-go-context` como un servidor MCP de contexto tecnico local y memoria operativa para desarrollo, manteniendo una identidad 100% Go:

- sin Python
- sin Node.js
- sin runtimes externos
- binario simple
- funcionamiento offline como prioridad

## Posicionamiento

`mcp-go-context` no debe competir como un sistema de memoria universal de largo plazo para cualquier tipo de dato.

Debe competir como:

- servidor MCP rapido y ligero
- analizador local de proyectos
- motor de contexto tecnico para asistentes de codigo
- memoria operativa para decisiones, debugging, onboarding y mantenimiento

## Principios del proyecto

1. Todo en Go.
2. Cero dependencias pesadas si no aportan una ventaja clara.
3. Offline first.
4. Mejorar precision antes que añadir muchas tools.
5. Mantener despliegue simple: un binario, una config, varios transportes.

## Estado actual

Base ya disponible:

- servidor MCP con transporte `stdio`, `http` y `sse`
- tools principales: `analyze-project`, `get-context`, `fetch-docs`, `remember-conversation`, `dependency-analysis`
- memoria persistente simple en JSON
- analisis de proyecto enfocado en Go
- configuracion JSON y cache local

## Fase 1: solidez y consistencia

Objetivo: corregir la base tecnica antes de expandir funcionalidad.

### Tareas

- Alinear imports internos con el modulo real del proyecto.
- Corregir `StoragePath` para que sea consistente con el uso real como directorio.
- Aplicar de verdad `MaxEntries` y `MaxResults` en memoria.
- Mejorar el orden de resultados de memoria por recencia y uso.
- Implementar limpieza basica por TTL y una politica simple de eviction.
- Revisar errores de configuracion y defaults para evitar estados ambiguos.

### Resultado esperado

- base estable
- configuracion coherente
- memoria utilizable en produccion
- menos deuda tecnica antes de crecer

## Fase 2: mejor contexto tecnico

Objetivo: mejorar la calidad del contexto que entrega el servidor.

### Tareas

- Añadir soporte para analizar dependencias de `package.json` desde Go.
- Añadir soporte para analizar `pyproject.toml` y `requirements.txt` desde Go, sin incorporar Python.
- Mejorar ranking de archivos relevantes en `get-context`.
- Pasar de contexto por archivo a contexto por simbolo o modulo cuando sea posible.
- Incorporar senales de Git para priorizar archivos cambiados recientemente.
- Crear una tool `changed-files-context` centrada en cambios recientes.

### Resultado esperado

- contexto mas preciso
- utilidad real fuera del ecosistema Go
- mejor posicionamiento como servidor MCP para proyectos modernos

## Fase 3: memoria operativa avanzada en Go

Objetivo: mejorar la memoria sin romper la identidad Go-only.

### Tareas

- Diseñar memoria hibrida opcional escrita en Go.
- Mantener almacenamiento simple, pero añadir mejor indexacion local.
- Mejorar busqueda de memoria con ranking por texto, tags, recencia y uso.
- Añadir tool `search-memory`.
- Añadir tool `save-decision` para decisiones tecnicas con estructura.
- Añadir trazabilidad de decisiones y fixes importantes.

### Resultado esperado

- memoria mas util para desarrollo real
- recuperacion mejor que substring plano
- ventaja competitiva sin depender de stacks externos

## Fase 4: producto y experiencia MCP

Objetivo: hacer que el proyecto se sienta mas completo como producto.

### Tareas

- Añadir tool `summarize-project` para resumen ejecutivo del repo.
- Añadir tool `explain-architecture` para describir modulos y relaciones.
- Mejorar respuestas de herramientas para que sean mas accionables.
- Añadir ejemplos de uso reales por workflow: debugging, onboarding, refactor, review.
- Refinar documentacion de instalacion, casos de uso y configuraciones por cliente.

### Resultado esperado

- mejor experiencia para usuarios finales
- propuesta de valor mas clara
- onboarding mas rapido

## Prioridad recomendada

### Prioridad alta

- Fase 1 completa
- ranking mejorado de contexto
- soporte multi-ecosistema de dependencias desde Go

### Prioridad media

- changed-files-context
- search-memory
- save-decision

### Prioridad baja

- explain-architecture
- summarize-project
- mejoras avanzadas de cache incremental

## Riesgos a evitar

- intentar replicar un sistema tipo MemPalace completo
- introducir Python o Node por rapidez
- añadir muchas tools sin mejorar precision
- crecer en marketing mas rapido que en fiabilidad

## Vision

La mejor version de `mcp-go-context` no es la mas grande.

Es la que resuelve mejor este problema:

> Dar a un asistente de codigo contexto tecnico local, util y rapido, con memoria operativa suficiente, usando solo Go.

## Siguiente paso recomendado

1. Ejecutar la Fase 1.
2. Abrir issues tecnicos concretos para cada tarea base.
3. Implementar despues `changed-files-context` y mejoras de ranking en `get-context`.
