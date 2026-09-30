---
title: Continuidad entre aplicaciones
description: Cómo retomar en otro cliente el trabajo guardado por Claude, OpenCode u otro asistente MCP.
---

La memoria compartida vive en el directorio configurado, por defecto `$HOME/.mcp-go-context/memory`. Los clientes del mismo equipo deben usar ese mismo directorio.

## Qué se comparte

| Información | Herramienta | Conservación |
|---|---|---|
| Decisiones y convenciones | `save-decision`, `promote-memory` | Duradera |
| Objetivo, pendientes y siguiente paso | `save-handoff` | Actualizable y con revisión |

Los archivos y Git siguen siendo la fuente de verdad del código. El resumen solo explica el estado del trabajo.

## Flujo

1. Al empezar trabajo sustantivo, llama a `resume-context` con la ruta del repositorio.
2. Pasa esa misma ruta a `save-decision` y `remember-conversation`.
3. Antes de cambiar de aplicación, guarda con `save-handoff` y la `expectedRevision` leída.
4. Si la respuesta contiene `CONFLICT`, conserva ambas versiones y reconcilialas.

Una consulta no escribe dentro del repositorio. La identidad del proyecto se guarda en el directorio de memoria. Abrir una subcarpeta resuelve la raíz Git cuando existe. Dos carpetas con el mismo nombre permanecen separadas.

## Instrucción para cada cliente

OpenCode usa `AGENTS.md`. Claude Code usa `CLAUDE.md`. En Claude Desktop, copia esta instrucción en las instrucciones del proyecto:

> Al empezar trabajo sustantivo, consulta `resume-context` con la ruta actual. Pasa esa ruta al guardar decisiones. Antes de cerrar o cambiar de cliente, llama a `save-handoff` con la revisión esperada. Si hay conflicto, conserva ambas versiones.

Reinicia los clientes después de actualizar el binario. No dejes un binario anterior escribiendo a la vez: no conoce los bloqueos nuevos.
