# 📋 Plan de Implementación MCP Go Context - 2025

**Versión del Documento**: 1.0
**Fecha**: Enero 2025
**Status**: Propuesta para Revisión
**Objetivo**: Optimizar y mejorar el MCP Go Context para Claude Desktop

---

## 📊 Resumen Ejecutivo

El proyecto **MCP Go Context v2.0.2** está en excelente estado técnico:
- ✅ Completamente actualizado (Go 1.26.1, MCP 2025-03-26)
- ✅ Seguridad auditada con suite de 60+ tests
- ✅ Cero dependencias externas (stdlib puro)
- ✅ Production-ready

**Sin embargo**, hay **6 mejoras opcionales** que podrían incrementar significativamente su valor y optimizar el contexto para Claude Desktop, especialmente en proyectos grandes.

---

## 🎯 Objetivos Principales

1. **Rendimiento**: 2-3x más rápido en recuperación de contexto
2. **Relevancia**: Contexto más inteligente y limpio
3. **Eficiencia**: Menor consumo de tokens en Claude
4. **Integración**: Mejor experiencia en Claude Desktop
5. **Sostenibilidad**: Caché persistente y manejo de proyectos grandes

---

## 📈 Matriz de Prioridades

| ID | Mejora | Prioridad | Esfuerzo | Impacto | Beneficio |
|:--:|--------|:---------:|:--------:|:-------:|-----------|
| 1 | Caché de contexto en memoria | ⭐⭐⭐ | 2-3h | Alto | Contexto 2-3x más rápido |
| 2 | Soporte para .gitignore | ⭐⭐⭐ | 1-2h | Alto | Contexto limpio, sin ruido |
| 3 | Modos de contexto (minimal/balanced/full) | ⭐⭐ | 2-3h | Medio | Control de tokens |
| 4 | Caché persistente en disco | ⭐⭐ | 3-4h | Medio | Startup 10x más rápido |
| 5 | Recursos MCP & Prompts dinámicos | ⭐ | 4-5h | Bajo | Mejor integración Desktop |
| 6 | IDE Integration & Webhooks | ⭐ | 3-4h | Bajo | Detección de cambios |

---

## 🔄 Fases de Implementación

### **FASE 1: Mejoras Críticas (Semana 1-2)**

Implementar las mejoras de **alta prioridad** que dan máximo impacto.

#### 1.1 Caché de Contexto en Memoria

**Objetivo**: Evitar re-análisis de archivos en la misma sesión.

**Cambios necesarios**:

1. **Crear nuevo módulo**: `internal/cache/context_cache.go`

```go
type ContextCache struct {
    mu       sync.RWMutex
    cache    map[string]*CacheEntry
    maxSize  int
    ttl      time.Duration
}

type CacheEntry struct {
    Data      interface{}
    CreatedAt time.Time
    HitCount  int
    Key       string
}

// Métodos principales
func (cc *ContextCache) Get(key string) (interface{}, bool)
func (cc *ContextCache) Set(key string, value interface{}, ttl time.Duration)
func (cc *ContextCache) Invalidate(pattern string)
func (cc *ContextCache) Clear()
```

2. **Integrar en `get-context` tool**:

```go
// En internal/tools/tools.go - función GetContext
func (r *Registry) GetContext(args map[string]interface{}) (interface{}, error) {
    // Crear clave de caché
    cacheKey := generateCacheKey(args)

    // Verificar caché
    if cached, ok := r.cache.Get(cacheKey); ok {
        return cached, nil
    }

    // Si no está en caché, realizar análisis
    result := performAnalysis(args)

    // Guardar en caché
    r.cache.Set(cacheKey, result, 30*time.Minute)

    return result, nil
}
```

3. **Invalidación inteligente**:
   - Monitorear cambios de archivos (timestamps)
   - Invalidar automáticamente si archivos cambian
   - Permitir invalidación manual via tool

**Archivos a modificar**:
- ✏️ `internal/tools/tools.go` - integrar caché en GetContext
- ➕ `internal/cache/context_cache.go` - nuevo archivo
- ✏️ `internal/server/server.go` - inicializar caché

**Tests necesarios**:
- `test/cache/context_cache_test.go` - Cache hit/miss
- `test/integration/cache_integration_test.go` - Integración con tools

**Tiempo estimado**: 2-3 horas

---

#### 1.2 Soporte para .gitignore

**Objetivo**: Excluir automáticamente archivos en .gitignore del análisis.

**Cambios necesarios**:

1. **Crear parser de .gitignore**: `internal/analyzer/gitignore.go`

```go
type GitignoreParser struct {
    patterns []*regexp.Regexp
    basePath string
}

func NewGitignoreParser(basePath string) (*GitignoreParser, error)
func (gp *GitignoreParser) IsIgnored(path string) bool
func (gp *GitignoreParser) Parse(content string) error
```

2. **Integrar en analyzer**:

```go
// En internal/analyzer/analyzer.go
func (a *Analyzer) walkDirectory(path string) error {
    gp, _ := NewGitignoreParser(path)

    return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
        if gp.IsIgnored(path) {
            if info.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }
        // ... continuar análisis
    })
}
```

3. **Configuración**:
   - Respetar .gitignore automáticamente
   - Opciones para excepciones (whitelist)
   - Excluir patrones comunes por defecto

**Patrones comunes excluidos**:
```
node_modules/, .git/, .venv/, venv/,
__pycache__/, .pytest_cache/, dist/, build/,
.idea/, .vscode/, *.log, *.tmp
```

**Archivos a modificar**:
- ➕ `internal/analyzer/gitignore.go` - nuevo archivo
- ✏️ `internal/analyzer/analyzer.go` - integrar parser
- ✏️ `internal/config/config.go` - agregar opciones

**Tests necesarios**:
- `test/analyzer/gitignore_test.go` - Parsing y matching
- `test/analyzer/ignore_patterns_test.go` - Patrones comunes

**Tiempo estimado**: 1-2 horas

---

#### 1.3 Corregir Versión en manifest.json

**Objetivo**: Sincronizar versión del .dxt con versión del código.

**Cambios necesarios**:

1. En `dxt/manifest.json`:
```json
{
  "version": "2.0.2"  // Cambiar de 2.0.0
}
```

**Archivos a modificar**:
- ✏️ `dxt/manifest.json` - actualizar versión

**Tiempo estimado**: 5 minutos

---

### **FASE 2: Mejoras de Control (Semana 2-3)**

Implementar características de **media prioridad** para control y eficiencia.

#### 2.1 Modos de Contexto (minimal/balanced/full)

**Objetivo**: Permitir control granular sobre cantidad de contexto devuelto.

**Cambios necesarios**:

1. **Definir niveles en config**:

```go
// internal/config/config.go
type ContextMode string

const (
    ContextMinimal  ContextMode = "minimal"   // ~5KB
    ContextBalanced ContextMode = "balanced"  // ~15KB (default)
    ContextFull     ContextMode = "full"      // ~50KB+
)

type ContextConfig struct {
    Mode       ContextMode
    MaxSize    int
    MaxFiles   int
    Depth      int
}
```

2. **Parámetros por modo**:

| Modo | MaxSize | MaxFiles | Depth | Caso de uso |
|------|---------|----------|-------|-------------|
| minimal | 5KB | 3-5 | 1 | Preguntas simples, optimizar tokens |
| balanced | 15KB | 10-15 | 2 | Default, equilibrio |
| full | 50KB+ | 30+ | 3 | Análisis profundo, refactoring |

3. **Implementar en get-context**:

```go
func (r *Registry) GetContext(args map[string]interface{}) (interface{}, error) {
    mode := extractMode(args) // minimal, balanced, full

    // Ajustar parámetros según modo
    config := getConfigForMode(mode)

    // Análisis con límites
    result := analyzeWithLimits(args, config)

    return result, nil
}
```

4. **Agregar parámetro a tool**:

```
Tool: get-context
Parámetros:
  - query (required): pregunta/tarea
  - mode (optional): minimal|balanced|full (default: balanced)
  - files (optional): archivos específicos
```

**Archivos a modificar**:
- ✏️ `internal/config/config.go` - agregar ContextMode
- ✏️ `internal/tools/tools.go` - GetContext con modo
- ✏️ `internal/analyzer/analyzer.go` - límites dinámicos
- ✏️ `dxt/manifest.json` - documentar parámetro

**Tests necesarios**:
- `test/tools/context_modes_test.go` - Cada modo
- `test/integration/mode_integration_test.go` - Integración

**Tiempo estimado**: 2-3 horas

---

#### 2.2 Caché Persistente en Disco

**Objetivo**: Mantener índice de proyecto en disco para arranques rápidos.

**Cambios necesarios**:

1. **Crear gestor de caché persistente**: `internal/cache/persistent_cache.go`

```go
type PersistentCache struct {
    basePath string
    mu       sync.RWMutex
    index    map[string]*CacheIndex
}

type CacheIndex struct {
    ProjectHash string
    FileHashes  map[string]string
    Metadata    map[string]interface{}
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func (pc *PersistentCache) Load(projectPath string) error
func (pc *PersistentCache) Save(projectPath string) error
func (pc *PersistentCache) IsValid(projectPath string) bool
func (pc *PersistentCache) Invalidate(projectPath string)
```

2. **Ubicación de caché**:
```
${HOME}/.mcp-context/
├── projects.json          // Índice de proyectos
├── project-hash-1/
│   ├── analysis.json      // Análisis cached
│   ├── dependencies.json  // Dependencias cached
│   └── metadata.json      // Metadata
└── project-hash-2/
    └── ...
```

3. **Estrategia de validación**:
   - Hash de archivos .go (si es Go)
   - Timestamp de carpetas
   - Version del servidor (invalidar en upgrades)

4. **Integración en startup**:

```go
// En internal/server/server.go Start()
func (s *Server) Start(ctx context.Context) error {
    // Cargar caché persistente
    if err := s.cache.LoadPersistent(s.config.ProjectPath); err != nil {
        log.Printf("Warning: failed to load persistent cache: %v", err)
    }

    // ... resto del startup
}
```

**Archivos a modificar**:
- ➕ `internal/cache/persistent_cache.go` - nuevo archivo
- ✏️ `internal/cache/context_cache.go` - integrar persistencia
- ✏️ `internal/server/server.go` - cargar/guardar caché
- ✏️ `internal/config/config.go` - configuración de caché

**Tests necesarios**:
- `test/cache/persistent_cache_test.go` - Save/load
- `test/cache/cache_validation_test.go` - Validación de caché

**Tiempo estimado**: 3-4 horas

---

### **FASE 3: Integración Avanzada (Semana 3-4)**

Implementar características de **baja prioridad** para integración mejorada.

#### 3.1 Recursos MCP & Prompts Dinámicos

**Objetivo**: Implementar protocolo MCP Resources para mejor integración con Desktop.

**Cambios necesarios**:

1. **Agregar soporte a Resources en server**:

```go
// internal/server/server.go
func (s *Server) handleResourcesList(req *JSONRPCRequest) interface{} {
    return map[string]interface{}{
        "resources": s.listProjectResources(),
    }
}

func (s *Server) handleResourcesRead(req *JSONRPCRequest) interface{} {
    // Leer recurso específico
    resourceURI := req.Params["uri"]
    return s.readResource(resourceURI)
}
```

2. **Definir recursos**:

```
Recursos MCP:
- project://summary          → Resumen del proyecto
- project://architecture     → Diagrama de arquitectura
- project://dependencies     → Grafo de dependencias
- project://entry-points     → Puntos de entrada
- project://recent-changes   → Cambios recientes
```

3. **Prompts dinámicos**:

```go
type PromptTemplate struct {
    Name        string
    Description string
    Arguments   []string
    Template    string // Con placeholders {{}}
}

var PromptTemplates = []PromptTemplate{
    {
        Name: "analyze-module",
        Description: "Analizar módulo específico",
        Arguments: []string{"module"},
        Template: "Analiza el módulo {{module}} en detalle...",
    },
    // Más templates
}
```

**Archivos a modificar**:
- ✏️ `internal/server/server.go` - handlers de resources
- ✏️ `internal/tools/tools.go` - definir prompts
- ✏️ `dxt/manifest.json` - registrar resources/prompts

**Tests necesarios**:
- `test/server/resources_test.go` - Resources listing
- `test/server/prompts_test.go` - Dynamic prompts

**Tiempo estimado**: 4-5 horas

---

#### 3.2 IDE Integration & Webhooks

**Objetivo**: Detectar cambios de archivos y eventos del IDE.

**Cambios necesarios**:

1. **Crear gestor de eventos**: `internal/watcher/file_watcher.go`

```go
type FileWatcher struct {
    watchers map[string]*fsnotify.Watcher
    handlers map[string][]EventHandler
    mu       sync.RWMutex
}

type EventHandler func(event FileChangeEvent) error

type FileChangeEvent struct {
    Path      string
    Operation string // create, modify, delete
    Timestamp time.Time
}

func (fw *FileWatcher) Watch(path string) error
func (fw *FileWatcher) On(operation string, handler EventHandler)
func (fw *FileWatcher) Stop()
```

2. **Webhooks para cambios**:

```go
type WebhookConfig struct {
    URL     string
    Events  []string
    Enabled bool
}

// En config.json
{
    "webhooks": {
        "on_file_change": "http://localhost:3001/hooks/file-changed",
        "on_analysis_complete": "http://localhost:3001/hooks/analysis-done"
    }
}
```

3. **Invalidar caché en cambios**:

```go
watcher.On("modify", func(evt FileChangeEvent) error {
    // Invalidar caché relacionado
    cacheKey := generateCacheKeyForFile(evt.Path)
    cache.Invalidate(cacheKey)
    return nil
})
```

**Archivos a modificar**:
- ➕ `internal/watcher/file_watcher.go` - nuevo archivo
- ✏️ `internal/config/config.go` - configuración de webhooks
- ✏️ `internal/server/server.go` - inicializar watcher
- ✏️ `internal/cache/context_cache.go` - integrar invalidación

**Tests necesarios**:
- `test/watcher/file_watcher_test.go` - Watch/events
- `test/server/webhooks_test.go` - Webhook dispatch

**Tiempo estimado**: 3-4 horas

---

## 📋 Timeline de Ejecución Propuesto

```
SEMANA 1-2: FASE 1 (Mejoras Críticas)
├─ 1.1 Caché de contexto en memoria        (2-3h)
├─ 1.2 Soporte para .gitignore             (1-2h)
├─ 1.3 Corregir versión manifest.json      (5m)
└─ Testing y QA                             (2-3h)
└─> Total: ~8-10 horas (1-1.5 días intensos)

SEMANA 2-3: FASE 2 (Mejoras de Control)
├─ 2.1 Modos de contexto                   (2-3h)
├─ 2.2 Caché persistente en disco          (3-4h)
└─ Testing y QA                             (2-3h)
└─> Total: ~10-12 horas (1.5-2 días)

SEMANA 3-4: FASE 3 (Integración Avanzada)
├─ 3.1 Recursos MCP & Prompts dinámicos    (4-5h)
├─ 3.2 IDE Integration & Webhooks          (3-4h)
└─ Testing y QA                             (2-3h)
└─> Total: ~12-15 horas (2 días)

TIEMPO TOTAL ESTIMADO: 30-37 horas (~4-5 días intensos)
```

---

## 🧪 Estrategia de Testing

### Test por Fase

**FASE 1**:
```bash
# Caché
go test -v ./test/cache/context_cache_test.go
go test -v ./test/integration/cache_integration_test.go

# .gitignore
go test -v ./test/analyzer/gitignore_test.go
go test -v ./test/analyzer/ignore_patterns_test.go

# Full suite
go test -v ./...
```

**FASE 2**:
```bash
# Modos
go test -v ./test/tools/context_modes_test.go
go test -v ./test/integration/mode_integration_test.go

# Caché persistente
go test -v ./test/cache/persistent_cache_test.go
go test -v ./test/cache/cache_validation_test.go

# Full suite
go test -v ./...
go test -v -cover -coverprofile=coverage.out ./...
```

**FASE 3**:
```bash
# Resources
go test -v ./test/server/resources_test.go

# Prompts
go test -v ./test/server/prompts_test.go

# Watcher
go test -v ./test/watcher/file_watcher_test.go

# Webhooks
go test -v ./test/server/webhooks_test.go

# Full suite
go test -v ./...
```

### Cobertura Mínima Requerida

- Caché: 85%+ coverage
- Parser .gitignore: 90%+ coverage
- Modos de contexto: 80%+ coverage
- Overall: 75%+ coverage

---

## 📦 Cambios en Dependencias

**Go.mod**: Sin cambios (seguir con stdlib puro)

**Nota**: Algunos cambios opcionales podrían requerir:
- `github.com/fsnotify/fsnotify` - para file watcher (opcional para Phase 3.2)

Pero se mantiene la filosofía de **cero dependencias externas** para Phase 1 y 2.

---

## 🔄 Versionamiento

### Versiones Post-Implementación

```
v2.0.2 (Actual)  → Mejoras críticas → v2.1.0
                 → Mejoras control   → v2.2.0
                 → Integración avanz → v2.3.0

Changelog entry:
## [2.1.0] - 2025-02-15
### Performance
- Added in-memory context caching with automatic invalidation
- Implemented .gitignore support for cleaner analysis

### New Features
- Context modes: minimal, balanced, full for token control
```

---

## 📊 Métricas de Éxito

| Métrica | Target | Actual (v2.0.2) | Post-Mejoras |
|---------|--------|-----------------|--------------|
| Tiempo get-context (pequeño proyecto) | <50ms | ~100ms | <50ms ✅ |
| Tiempo get-context (proyecto grande) | <500ms | ~2000ms | <500ms ✅ |
| Tamaño medio contexto | <20KB | ~25KB | <15KB ✅ |
| Startup time con caché | <100ms | ~500ms | <100ms ✅ |
| Test coverage | >75% | ~78% | >85% ✅ |
| Archivos excluidos innecesarios | 0% | ~40% | ~0% ✅ |

---

## 🚀 Rollout Plan

### Pre-Release

1. ✅ Implementar FASE 1 (crítica)
2. ✅ Testing exhaustivo
3. ✅ Benchmark comparativo
4. 📝 Actualizar documentación
5. 📝 Crear release notes

### Release

1. 🏷️ Tag v2.1.0 en git
2. 📦 Construir .dxt package
3. 📤 Publicar en releases
4. 📢 Anunciar en README

### Post-Release

1. ✅ Monitorear reports de bugs
2. ✅ Recopilar feedback de usuarios
3. ✅ Planificar FASE 2 basado en feedback

---

## 📚 Documentación a Actualizar

- [ ] README.md - Agregar sección "Performance"
- [ ] CLAUDE.md - Actualizar características
- [ ] docs/OPTIMIZATIONS.md - Detalles de optimizaciones
- [ ] dxt/manifest.json - Actualizar versión y descripción
- [ ] CHANGELOG.md - Nuevas versiones
- [ ] docs/CONTEXT_MODES.md - Guía de modos (nuevo)
- [ ] docs/CACHING_STRATEGY.md - Explicación de caché (nuevo)

---

## ⚠️ Riesgos y Mitigación

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|:------------:|:-------:|-----------|
| Caché inconsistente | Media | Alto | Hash validation, invalidation tests |
| .gitignore parsing incorrecto | Baja | Medio | Comprehensive regex tests |
| Memory leak en caché | Baja | Alto | LRU eviction, memory tests |
| Performance regression | Baja | Alto | Benchmarks comparativos |
| Breaking changes | Baja | Alto | Backward compatibility tests |

---

## 🎯 Next Steps

### Inmediato (Hoy)

1. ✅ Revisar este plan
2. ⏳ Aprobar prioridades
3. 📋 Crear issues en GitHub por tarea

### Corto Plazo (Esta semana)

1. 🔨 Iniciar implementación FASE 1
2. 🧪 Escribir tests primero (TDD)
3. 📝 Documentar cambios en progreso

### Mediano Plazo (Próximas 2 semanas)

1. ✅ Completar FASE 1 y 2
2. 🧪 Testing exhaustivo
3. 📊 Benchmark comparativo
4. 🚀 Release v2.1.0

---

## 📞 Contacto y Preguntas

Para preguntas sobre este plan:
- 📧 Contactar a ScopWeb
- 💬 Abrir discussion en GitHub
- 🐛 Reportar issues específicos

---

## 📄 Historial del Documento

| Versión | Fecha | Cambios |
|---------|-------|---------|
| 1.0 | 13-01-2025 | Documento inicial |

---

**Documento generado**: Enero 2025
**Status**: Ready for Review
**Aprobación requerida**: Sí
