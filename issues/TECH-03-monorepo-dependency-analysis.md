# Issue TEC-03: mejorar dependency-analysis para monorepos y workspaces mixtas

**Etiquetas**: `enhancement`, `analysis`, `monorepo`, `priority-high`, `go-only`

## Problema

El analisis actual revisa manifests de forma demasiado plana y centrada en un solo `ProjectPaths[0]`. Eso no refleja bien workspaces con multiples apps, paquetes o modulos.

## Impacto

- contexto de dependencias incompleto o ambiguo
- baja utilidad en repos TypeScript y Python mixtos
- peor ranking de contexto en repositorios grandes

## Objetivo

Convertir `dependency-analysis` en una tool fiable para repos con multiples subproyectos, sin salir de Go.

## Alcance

### Cambios esperados

1. Iterar todos los `ProjectPaths` configurados.
2. Detectar manifests por recorrido controlado dentro de cada raiz.
3. Soportar multiples `go.mod`, `package.json`, `pyproject.toml` y `requirements.txt`.
4. Asociar cada dependencia a un scope claro.

Scopes minimos:

- `root`
- `module`
- `package`
- `app` o `service`

5. Mejorar parser de `pyproject.toml` sin introducir dependencias pesadas salvo necesidad clara.
6. Valorar soporte de `pnpm-workspace.yaml` si el coste sigue bajo.

### Fuera de alcance

- resolucion completa de grafos npm o python
- interpretacion de lockfiles complejos en esta primera iteracion

## Archivos probables

- `internal/analyzer/analyzer.go`
- `internal/config/config.go`

## Criterios de aceptacion

- el analisis refleja multiples subproyectos reales
- la salida distingue dependencias por scope
- no se mezclan dependencias de forma ambigua
- hay tests para un monorepo mixto

## Definicion de terminado

`dependency-analysis` resulta util para una workspace con proyectos Go, TypeScript y Python coexistiendo.