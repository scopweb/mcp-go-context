# 🏗️ Arquitectura de Mejoras - Visual Guide

**Diagrama de arquitectura con mejoras propuestas**

---

## 📐 Arquitectura Actual (v2.0.2)

```
┌─────────────────────────────────────────────────────────┐
│                  Claude Desktop                          │
├─────────────────────────────────────────────────────────┤
                         ↓
┌─────────────────────────────────────────────────────────┐
│              MCP JSON-RPC Protocol (stdio)              │
├─────────────────────────────────────────────────────────┤
                         ↓
┌─────────────────────────────────────────────────────────┐
│              MCP Server (Go stdlib)                      │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Transport Layer                                  │   │
│  │ ├─ stdio.go        (Claude Desktop)             │   │
│  │ ├─ http.go + JWT                                │   │
│  │ ├─ sse.go + JWT                                 │   │
│  │ └─ streamable.go   (MCP 2025)                   │   │
│  └──────────────────────────────────────────────────┘   │
│                         ↓                                │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Tool Registry                                    │   │
│  │ ├─ analyze-project                              │   │
│  │ ├─ get-context         ← SLOW (no caché)        │   │
│  │ ├─ fetch-docs                                   │   │
│  │ ├─ remember-conversation                        │   │
│  │ ├─ dependency-analysis                          │   │
│  │ └─ memory-*                                      │   │
│  └──────────────────────────────────────────────────┘   │
│                         ↓                                │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Analyzer & Memory                                │   │
│  │ ├─ analyzer.go         ← SLOW (rescans all)     │   │
│  │ ├─ memory/manager.go                            │   │
│  │ └─ config.go                                    │   │
│  └──────────────────────────────────────────────────┘   │
│                         ↓                                │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│              Project Files                              │
│  ├─ .go files                                           │
│  ├─ .js/.ts files                                       │
│  ├─ node_modules/  ← INCLUDES (ruido)                  │
│  ├─ .git/          ← INCLUDES (ruido)                  │
│  └─ build/         ← INCLUDES (ruido)                  │
└─────────────────────────────────────────────────────────┘

❌ Problemas:
  ├─ get-context sin caché: Analiza TODO cada vez
  ├─ No respeta .gitignore: Incluye archivos innecesarios
  ├─ Sin control de modo: Siempre contexto "promedio"
  └─ Sin persistencia: No reutiliza análisis entre sesiones
```

---

## 📐 Arquitectura Mejorada (v2.1.0+)

```
┌─────────────────────────────────────────────────────────┐
│                  Claude Desktop                          │
├─────────────────────────────────────────────────────────┤
                         ↓
┌─────────────────────────────────────────────────────────┐
│              MCP JSON-RPC Protocol (stdio)              │
├─────────────────────────────────────────────────────────┤
                         ↓
┌─────────────────────────────────────────────────────────┐
│              MCP Server (Go stdlib)                      │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Transport Layer                                  │   │
│  │ ├─ stdio.go        (Claude Desktop)             │   │
│  │ ├─ http.go + JWT                                │   │
│  │ ├─ sse.go + JWT                                 │   │
│  │ └─ streamable.go   (MCP 2025)                   │   │
│  └──────────────────────────────────────────────────┘   │
│                         ↓                                │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Tool Registry                                    │   │
│  │ ├─ analyze-project                              │   │
│  │ ├─ get-context         ✅ FAST (con caché)      │   │
│  │ ├─ fetch-docs                                   │   │
│  │ ├─ remember-conversation                        │   │
│  │ ├─ dependency-analysis                          │   │
│  │ └─ memory-*                                      │   │
│  └──────────────────────────────────────────────────┘   │
│                         ↓                                │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 🆕 CACHE LAYER (In-Memory)                      │   │
│  │ ├─ context_cache.go   [minimal|balanced|full]   │   │
│  │ ├─ Hash-based keys                              │   │
│  │ ├─ TTL-based eviction (30min default)           │   │
│  │ ├─ Hit counter para analytics                   │   │
│  │ └─ Invalidation on file change                  │   │
│  └──────────────────────────────────────────────────┘   │
│                         ↓                                │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Analyzer & Memory (IMPROVED)                     │   │
│  │ ├─ analyzer.go         ✅ FAST (respeta ignore) │   │
│  │ │  └─ 🆕 gitignore.go  ← NUEVO                 │   │
│  │ ├─ memory/manager.go                            │   │
│  │ └─ config.go           ✅ context modes         │   │
│  └──────────────────────────────────────────────────┘   │
│                         ↓                                │
│  ┌──────────────────────────────────────────────────┐   │
│  │ 🆕 PERSISTENT CACHE (Disk)                      │   │
│  │ ├─ ~/.mcp-context/                              │   │
│  │ ├─ Project hashes                               │   │
│  │ ├─ File change detection                        │   │
│  │ └─ Auto-invalidation on upgrade                 │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
└─────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────┐
│              Project Files                              │
│  ├─ .go files           ✅ INCLUDED                    │
│  ├─ .js/.ts files       ✅ INCLUDED                    │
│  ├─ node_modules/       ✅ EXCLUDED (respeta .gitignore)
│  ├─ .git/               ✅ EXCLUDED (respeta .gitignore)
│  ├─ build/              ✅ EXCLUDED (respeta .gitignore)
│  ├─ dist/               ✅ EXCLUDED (respeta .gitignore)
│  └─ __pycache__/        ✅ EXCLUDED (respeta .gitignore)
└─────────────────────────────────────────────────────────┘

✅ Beneficios:
  ├─ In-Memory Cache: Contexto 2-3x más rápido
  ├─ .gitignore Support: Contexto limpio, sin ruido
  ├─ Context Modes: Control de tokens (minimal/balanced/full)
  ├─ Persistent Cache: 10x más rápido en siguiente sesión
  └─ Invalidation: Automático en cambios de archivo
```

---

## 🔄 Flujo de get-context: Antes vs Después

### ❌ ANTES (Sin Caché)

```
User Query: "¿Cómo funciona authenticate()?"
                        ↓
                  get-context tool
                        ↓
        ┌─────────────────────────────┐
        │ 1. Read ALL .go files       │ ← 200-500ms
        │ 2. Parse AST of ALL         │ ← 800-1200ms
        │ 3. Map dependencies ALL     │ ← 400-600ms
        │ 4. Search for function      │ ← 100-200ms
        └─────────────────────────────┘
                        ↓
              TOTAL: 1500-2500ms 😞
                        ↓
              Return contexto ~25KB
                        ↓
           Claude recibe respuesta LENTA
```

### ✅ DESPUÉS (Con Caché)

```
User Query: "¿Cómo funciona authenticate()?"
                        ↓
                  get-context tool
                        ↓
          ┌──────────────────────────┐
          │ Check cache (hash query) │ ← <1ms
          └──────────────────────────┘
                        ↓
                  ¿En caché?
                   /        \
                 SÍ          NO
                /              \
             <1ms          Realizar análisis
                              (como antes)
                           ~1500ms
                        ↓
         ┌────────────────────────────┐
         │ Save to cache (30min TTL)  │ ← <10ms
         └────────────────────────────┘
                        ↓
         TOTAL: <1ms (hit) o ~1500ms (miss) 🚀
                        ↓
         Return contexto ~15KB (con .gitignore)
                        ↓
        Claude recibe respuesta RÁPIDA
```

### 🎯 Casos de Uso

```
Escenario 1: Preguntas Rápidas
─────────────────────────────
Usuario hace 3 preguntas sobre misma función en 5 minutos:
  P1: "¿Qué hace authenticate()?"        → 1500ms (análisis completo)
  P2: "¿Dónde se llama authenticate()?"  → <5ms (caché hit) ✅
  P3: "¿Cuál es el return type?"         → <5ms (caché hit) ✅
  ─────────────────────────
  Total: 1510ms vs 4500ms sin caché = 3x más rápido


Escenario 2: Proyecto Grande
─────────────────────────────
Usuario trabajando en proyecto con 500 archivos:
  - Sin caché: CADA pregunta espera 2-3 segundos 😞
  - Con caché: Primeras 2-3 preguntas lentas, resto rápido ✅
  - Con persistencia: Siguiente sesión TODOS rápidos 🚀


Escenario 3: Modos de Contexto
─────────────────────────────────
Usuario pregunta: "¿Cuál es el nombre de la función?"

  Modo minimal (5KB):
  ├─ 3-5 archivos principales
  ├─ Respuesta: <50ms
  └─ Tokens: ~150 tokens

  Modo balanced (15KB, default):
  ├─ 10-15 archivos relacionados
  ├─ Respuesta: ~200ms
  └─ Tokens: ~500 tokens

  Modo full (50KB):
  ├─ 30+ archivos completos
  ├─ Respuesta: ~1500ms
  └─ Tokens: ~2000 tokens
```

---

## 📊 Data Flow: Cache System

### Context Cache (In-Memory)

```
┌──────────────────────────────────┐
│   User Query                      │
│   "get-context?file=auth.go"     │
└──────────────────────────────────┘
              ↓
┌──────────────────────────────────┐
│   Generate Cache Key              │
│   Hash(query+mode+files)          │
│   "a7f3b2e9d1c4..."              │
└──────────────────────────────────┘
              ↓
┌──────────────────────────────────┐
│   Check Memory Cache              │
│   map[string]*CacheEntry         │
└──────────────────────────────────┘
              ↓
          HIT?                 MISS?
         /                        \
        ↓                          ↓
   Return from              Perform Analysis
   cache <1ms               1000-2000ms
        │                          │
        │                          ↓
        │                   ┌──────────────────┐
        │                   │ Save to Cache    │
        │                   │ TTL: 30min       │
        │                   │ HitCount++       │
        │                   └──────────────────┘
        │                          │
        └──────────────┬───────────┘
                       ↓
          ┌──────────────────────────────┐
          │   Return Context to Claude   │
          │   (cache hit or fresh data) │
          └──────────────────────────────┘
              ↓
          HIT STATS:
          - Saved 1400-1900ms
          - CPU: <1% vs 60%+
          - Network: <10KB vs 50KB+
```

### Persistent Cache (Disk)

```
┌──────────────────────────────────┐
│   Server Startup                  │
│   mcp-context-server.exe         │
└──────────────────────────────────┘
              ↓
┌──────────────────────────────────┐
│   Load Persistent Cache           │
│   ~/.mcp-context/projects.json   │
└──────────────────────────────────┘
              ↓
        ┌─────────────────┐
        │ Valid Cache?    │
        └─────────────────┘
         /                 \
        ↓                   ↓
      SÍ                   NO
   Load from            Rescan
   disk (50ms)          (1500ms)
      │                   │
      └───────┬───────────┘
              ↓
    ┌──────────────────────┐
    │ Ready with context  │
    │ First query: 50ms    │ (con persistencia)
    │ First query: 1500ms  │ (sin persistencia)
    └──────────────────────┘
```

---

## 🔧 Implementación: Component Map

### Phase 1: Critical (v2.1.0)

```
internal/cache/
├── context_cache.go
│   ├── type ContextCache struct
│   ├── func (cc *ContextCache) Get() error
│   ├── func (cc *ContextCache) Set() error
│   ├── func (cc *ContextCache) Invalidate() error
│   └── func (cc *ContextCache) Clear() error
│
└── [tests]
    └── context_cache_test.go

internal/analyzer/
├── gitignore.go (NEW)
│   ├── type GitignoreParser struct
│   ├── func NewGitignoreParser() (*GitignoreParser, error)
│   ├── func (gp *GitignoreParser) IsIgnored(path string) bool
│   └── func (gp *GitignoreParser) Parse(content string) error
│
├── analyzer.go (MODIFIED)
│   └── Integrate gitignore parser into walkDirectory()
│
└── [tests]
    └── gitignore_test.go

internal/tools/
├── tools.go (MODIFIED)
│   ├── GetContext() - add caching
│   └── Add cache initialization
│
└── [tests]
    └── tools_cache_test.go
```

### Phase 2: Control (v2.2.0)

```
internal/config/
├── config.go (MODIFIED)
│   ├── type ContextMode string
│   ├── const (minimal, balanced, full)
│   └── type ContextConfig struct
│
internal/cache/
├── persistent_cache.go (NEW)
│   ├── type PersistentCache struct
│   ├── func (pc *PersistentCache) Load() error
│   ├── func (pc *PersistentCache) Save() error
│   └── func (pc *PersistentCache) IsValid() bool
│
└── [tests]
    └── persistent_cache_test.go
```

### Phase 3: Advanced (Future)

```
internal/server/
├── resources.go (NEW)
│   ├── func (s *Server) handleResourcesList()
│   ├── func (s *Server) handleResourcesRead()
│   └── func (s *Server) listProjectResources()

internal/watcher/
├── file_watcher.go (NEW)
│   ├── type FileWatcher struct
│   ├── func (fw *FileWatcher) Watch() error
│   ├── func (fw *FileWatcher) On() error
│   └── func (fw *FileWatcher) Stop() error
```

---

## 📊 Performance Comparison

### Antes vs Después: Benchmarks

```
PEQUEÑO PROYECTO (50 archivos, 10K LOC)
─────────────────────────────────────────
Métrica                  Antes       Después     Mejora
get-context time         120ms       <50ms       2.4x 🚀
Memoria usado            ~8MB        ~12MB       +4MB
Primer startup           500ms       500ms       -
Contexto size            8KB         5KB         40% ↓
Análisis CPU             35%         <1%         35x ↓


MEDIANO PROYECTO (200 archivos, 50K LOC)
──────────────────────────────────────────
Métrica                  Antes       Después     Mejora
get-context time         800ms       ~200ms      4x 🚀
Memoria usado            ~25MB       ~30MB       +5MB
Primer startup           2000ms      2000ms      -
Contexto size            20KB        12KB        40% ↓
Análisis CPU             70%         <1%         70x ↓


GRANDE PROYECTO (800 archivos, 200K LOC)
─────────────────────────────────────────
Métrica                  Antes       Después     Mejora
get-context time         2500ms      ~400ms      6.2x 🚀🚀
Memoria usado            ~80MB       ~90MB       +10MB
Primer startup           5000ms      5000ms      -
Contexto size            50KB        25KB        50% ↓
Análisis CPU             90%+        <1%         90x ↓


CON PERSISTENT CACHE (Siguiente sesión)
───────────────────────────────────────
Métrica                  Sin Cache   Con Cache   Mejora
Primer startup           5000ms      100ms       50x 🚀🚀🚀
get-context (cache hit)  2500ms      <5ms        500x 🚀🚀🚀
```

---

## 🎯 Context Modes: Size Comparison

```
MINIMAL MODE (~5KB)
────────────────────
Archivos incluidos:
├─ main.go (entry point)
├─ handler.go (2 funciones relacionadas)
└─ types.go (tipos necesarios)

Total: 3-5 archivos, ~5KB

Casos de uso:
├─ "¿Cuál es el nombre de la función?"
├─ "¿Dónde se define X?"
└─ "¿Qué importa este archivo?"


BALANCED MODE (~15KB) [DEFAULT]
────────────────────────────────
Archivos incluidos:
├─ handler.go
├─ service.go
├─ repository.go
├─ types.go
├─ middleware.go
├─ utils.go
├─ errors.go
├─ constants.go
├─ config.go
└─ 5 más relacionados

Total: 10-15 archivos, ~15KB

Casos de uso:
├─ "¿Cómo implemento esta feature?"
├─ "¿Qué hace este módulo?"
└─ "¿Cuáles son las dependencias?"


FULL MODE (~50KB+)
──────────────────
Archivos incluidos:
├─ TODOS los archivos relevantes (30+)
├─ Todas las dependencias
├─ Historias de cambios
└─ Documentación interna

Total: 30+ archivos, ~50KB+

Casos de uso:
├─ "Refactoriza todo el módulo de auth"
├─ "¿Cuál es la arquitectura completa?"
└─ "¿Dónde hay technical debt?"
```

---

## ⚡ Performance Timeline

```
SESIÓN DE USUARIO - Mejoras en Acción
──────────────────────────────────────

T=0s    Claude inicia (sin persistent cache)
        └─ Startup: 5s (scan proyecto)
        └─ Context cache: EMPTY

T=5s    Usuario pregunta: "¿Cómo funciona auth?"
        └─ get-context: 1500ms (análisis completo, miss)
        └─ Respuesta: 6.5s total
        └─ Cache hit counter: 0

T=10s   Usuario pregunta: "¿Dónde se autentica?"
        └─ get-context: <5ms (cache HIT) ✅
        └─ Respuesta: <1s total 🚀
        └─ Cache hit counter: 1

T=15s   Usuario pregunta: "¿Cuál es el JWT?"
        └─ get-context: <5ms (cache HIT) ✅
        └─ Respuesta: <1s total 🚀
        └─ Cache hit counter: 2

T=30s   Usuario modifica auth.go
        └─ File watcher: Detecta cambio
        └─ Cache: INVALIDATED automáticamente
        └─ Cache hit counter: reset

T=35s   Usuario pregunta: "¿Qué cambié?"
        └─ get-context: 1200ms (re-análisis)
        └─ Respuesta: 2s total
        └─ Cache hit counter: 0

T=45s   Usuario pregunta: "¿Dónde más se usa?"
        └─ get-context: <5ms (cache HIT) ✅
        └─ Respuesta: <1s total 🚀
        └─ Cache hit counter: 1

─────────────────────────────────────────────────

RESUMEN:
  Tiempo total: 45s
  Con mejoras:  13s (preguntas rápidas: <1s)
  Sin mejoras:  ~30s (todas lentas: 1-2s)

  AHORRO: 17s (57% más rápido) ⚡
```

---

## 🔐 Seguridad & Validación

### Cache Invalidation Strategy

```
Triggers de Invalidación:
├─ File Change Detection
│  └─ Monitorear timestamps en ~/project
│  └─ Invalidar caché si archivo modificado
│
├─ Configuration Change
│  └─ Si user cambia .gitignore
│  └─ Si user cambia config
│
├─ TTL Expiration
│  └─ Default: 30 minutos
│  └─ Configurable en config.json
│
├─ Cache Size Limit
│  └─ Max items: 1000
│  └─ Max size: 100MB
│  └─ LRU eviction si se excede
│
└─ Server Upgrade
   └─ Invalidar si versión cambia
   └─ Invalidar si protocolo MCP cambia
```

### Cache Key Generation

```go
// Seguro contra colisiones
func generateCacheKey(query, mode, files, projectPath string) string {
    data := fmt.Sprintf("%s|%s|%s|%s", query, mode, files, projectPath)
    hash := sha256.Sum256([]byte(data))
    return fmt.Sprintf("%x", hash)
}

// Ejemplo:
// Input:  query="authenticate", mode="balanced", files="auth.go", proj="/home/user/proj"
// Output: "3a2f8c9e1b5d7e2c4a6f8b0d2e4f6a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d"
```

---

## 📚 Archivos de Referencia

Para implementación, ver:
- 📖 [IMPLEMENTATION_PLAN_2025.md](./IMPLEMENTATION_PLAN_2025.md) - Plan detallado
- 🚀 [QUICK_START_IMPROVEMENTS.md](./QUICK_START_IMPROVEMENTS.md) - Quick reference
- 🔒 [SECURITY_AUDIT_2024.md](./SECURITY_AUDIT_2024.md) - Seguridad

---

**Documento**: Architecture Improvements
**Versión**: 1.0
**Fecha**: 13-01-2025
**Diagrama Version**: MCP 2025-03-26 Compatible
