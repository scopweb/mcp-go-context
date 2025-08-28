# 🔒 JWT Security Implementation Guide

## ✅ Implementación Completada

### Nuevas Características de Seguridad

1. **Autenticación JWT** reemplaza la autenticación simple por token
2. **Configuración de seguridad** completa y flexible
3. **Validación de tokens** con expiración y verificación de firma
4. **Herramienta de generación** de tokens para desarrollo

### Archivos Modificados/Creados

- **`internal/auth/jwt.go`** - Nuevo módulo de autenticación JWT
- **`internal/config/config.go`** - Configuración de seguridad extendida  
- **`internal/server/server.go`** - Integración JWT en el servidor
- **`test/auth_jwt_simple_test.go`** - Tests de autenticación JWT
- **`examples/jwt-config.json`** - Configuración de ejemplo

## 🚀 Cómo Usar JWT Authentication

### 1. Habilitar JWT

```bash
# Establecer el secreto JWT (requerido)
export MCP_JWT_SECRET="tu-secreto-super-seguro-aqui"

# O usar archivo de configuración
```

### 2. Configuración JSON

```json
{
  "security": {
    "auth": {
      "enabled": true,
      "method": "jwt",
      "secret": "", 
      "expiry": "1h",
      "issuer": "mcp-go-context",
      "algorithm": "HS256"
    }
  }
}
```

### 3. Generar Token de Desarrollo

```bash
# Ejecutar el servidor con JWT habilitado
./bin/mcp-context-server.exe --transport http --port 3000

# Usar la herramienta auth-generate-token
curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "auth-generate-token",
      "arguments": {
        "subject": "mi-usuario"
      }
    }
  }'
```

### 4. Hacer Requests Autenticados

```bash
# HTTP Request con JWT
curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN_HERE" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize"
  }'
```

## 🛡️ Características de Seguridad

### Validaciones Implementadas

- ✅ **Verificación de firma HMAC-SHA256**
- ✅ **Validación de expiración** de tokens
- ✅ **Verificación de emisor** (issuer)
- ✅ **Validación de formato** JWT
- ✅ **Protección contra replay** con JTI único
- ✅ **Solo HTTP/SSE** - stdio no requiere auth

### Configuraciones de Seguridad

```json
{
  "security": {
    "cors": {
      "enabled": true,
      "origins": ["https://localhost:3000", "app://claude-desktop"]
    },
    "rateLimit": {
      "enabled": false,
      "requests": 100,
      "window": "1m"
    }
  }
}
```

## 🧪 Tests Implementados

```bash
# Ejecutar tests de JWT
go test -v ./test -run TestJWTAuthSimple

# Todos los tests
go test -v ./...
```

### Casos de Prueba Cubiertos

- ✅ Generación y validación de tokens
- ✅ Manejo de tokens expirados
- ✅ Extracción de headers de autorización
- ✅ Configuración del servidor con JWT
- ✅ Rechazo de tokens inválidos

## ⚠️ Consideraciones de Producción

### Variables de Entorno Requeridas

```bash
# OBLIGATORIO para JWT
export MCP_JWT_SECRET="secreto-muy-seguro-64-caracteres-minimo"

# Opcional - configuración adicional
export MCP_CONFIG_PATH="./config.json"
```

### Recomendaciones de Seguridad

1. **Secreto seguro**: Mínimo 64 caracteres, aleatorio
2. **HTTPS obligatorio** en producción (próxima implementación)
3. **Rotación de secretos** periódica
4. **Monitoreo de logs** de seguridad
5. **Rate limiting** habilitado en producción

## 📋 Próximos Pasos (Pendientes)

Para completar las 5 mejoras de seguridad:

2. **CORS Configurables** ✅ (Implementado en configuración)
3. **HTTPS Obligatorio** (Pendiente)
4. **Desktop Extensions (.dxt)** (Pendiente) 
5. **Streamable HTTP Transport** (Pendiente)

## 🎯 Compatibilidad

- ✅ **Claude Desktop** - Funciona con stdio (sin auth)
- ✅ **HTTP Clients** - Requiere JWT con `Authorization: Bearer`
- ✅ **SSE Clients** - Requiere JWT en requests
- ✅ **Backward Compatible** - stdio sigue funcionando sin cambios

La implementación JWT está **completa y testeada**. El servidor ahora ofrece autenticación moderna y segura para conexiones HTTP/SSE mientras mantiene compatibilidad con stdio para Claude Desktop.