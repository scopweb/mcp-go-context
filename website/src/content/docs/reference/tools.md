---
title: Referencia de herramientas
description: Documentación completa de todas las herramientas MCP disponibles.
---

## MCP Tools

El servidor expone las siguientes herramientas a través del protocolo MCP.

### analyze-project

Analyzes project structure, dependencies, and provides comprehensive context.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `path` | string | Project path to analyze (default: current directory) |
| `depth` | integer | Analysis depth (default: 3) |

**Ejemplo:**
```
Use analyze-project with path="." and depth=3
```

---

### get-context

Retrieves relevant context for the current task based on files, dependencies, and conversation history.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `query` | string | Context query or topic (required) |
| `files` | array | Specific files to include in context |
| `maxTokens` | integer | Maximum tokens to return (default: 10000) |

**Ejemplo:**
```
Use get-context with query="database migrations" and maxTokens=8000
```

---

### fetch-docs

Fetches documentation for libraries and dependencies via Context7 with local fallbacks.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `library` | string | Library name to fetch docs for (required) |
| `version` | string | Specific version (optional) |
| `topic` | string | Specific topic within the docs (optional) |

**Ejemplo:**
```
Use fetch-docs with library="gin-gonic/gin" and topic="routing"
```

---

### dependency-analysis

Analyzes project dependencies and suggests relevant documentation.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `includeTransitive` | boolean | Include transitive dependencies |
| `onlyDirect` | boolean | Only analyze direct dependencies |
| `suggestDocs` | boolean | Suggest documentation URLs for dependencies |

**Ejemplo:**
```
Use dependency-analysis with includeTransitive=true and suggestDocs=true
```

---

### changed-files-context

Gets context from files changed in recent git commits.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `path` | string | Project path (default: current directory) |
| `commitCount` | integer | Number of recent commits to analyze (default: 5) |
| `maxFiles` | integer | Maximum number of files to include (default: 10) |

**Ejemplo:**
```
Use changed-files-context with commitCount=10 and maxFiles=20
```

---

### remember-conversation

Stores important context from the current conversation for future reference.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `key` | string | Key to store the memory under (required) |
| `content` | string | Content to remember (required) |
| `tags` | array | Tags for categorization |

**Ejemplo:**
```
Use remember-conversation with key="auth-bug-fix", content="The authentication bug was in the token validation logic...", and tags=["bug", "security"]
```

---

### save-decision

Records a technical decision with structured metadata for future reference.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `key` | string | Unique identifier for this decision (required) |
| `content` | string | Description of the decision made (required) |
| `decisionType` | string | Type: architecture, fix, approach, tech-debt, security, performance |
| `reason` | string | Why this decision was made |
| `alternatives` | array | Alternative approaches that were considered |
| `tags` | array | Additional tags |

**Ejemplo:**
```
Use save-decision with key="use-postgres", content="Chose PostgreSQL over MongoDB...", decisionType="architecture", reason="Better ACID compliance for financial transactions", and alternatives=["mongodb", "mysql"]
```

---

### get-decisions

Retrieves technical decisions, optionally filtered by type or keyword.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `decisionType` | string | Filter by decision type |
| `keyword` | string | Search keyword in decision content |
| `limit` | integer | Maximum number of results (default: 10) |

**Ejemplo:**
```
Use get-decisions with decisionType="architecture" and limit=10
```

---

### promote-memory

Marks a memory as promoted (high-value) for persistent storage.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `key` | string | Memory key to promote (required) |
| `confidence` | string | Confidence level: low, medium, high (default: high) |

**Ejemplo:**
```
Use promote-memory with key="auth-bug-fix" and confidence="high"
```

---

### get-promoted-memories

Returns memories that have been promoted from session memory.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `limit` | integer | Maximum number of results (default: 10) |

**Ejemplo:**
```
Use get-promoted-memories with limit=25
```

---

### search-memory

Advanced search through conversation memory with ranking by relevance, recency, and usage.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `query` | string | Search query to match against memory content |
| `tags` | array | Filter by tags |
| `limit` | integer | Maximum number of results (default: 10) |

**Ejemplo:**
```
Use search-memory with query="authentication" and tags=["security", "api"]
```

---

### suggest-promotions

Analiza tus memorias existentes usando múltiples señales de calidad (estructura de decisión, frecuencia de uso, recencia, lenguaje de decisión, etc.) y devuelve las mejores candidatas para promover con `promote-memory`.

**Esta es la herramienta más importante** para mantener una memoria de alto valor sin llenarla de ruido.

**Parámetros:**

| Parámetro | Tipo | Descripción |
|----------|------|-------------|
| `limit` | integer | Número máximo de sugerencias (por defecto: 5) |

**Flujo recomendado:**
1. Terminas una sesión compleja
2. Ejecutas `suggest-promotions`
3. Revisas las sugerencias
4. Promueves solo lo que realmente quieres conservar a largo plazo con `promote-memory`