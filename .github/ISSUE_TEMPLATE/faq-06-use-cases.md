---
name: "❓ FAQ - ¿Cuáles son los casos de uso prácticos del MCP?"
about: Pregunta frecuente sobre casos de uso reales y prácticos del MCP
title: "❓ FAQ - ¿Cuáles son los casos de uso prácticos del MCP?"
labels: documentation, use-cases, examples, workflow, faq
assignees: ''
---

# ❓ Pregunta Frecuente: "¿Cuáles son los casos de uso prácticos del MCP?"

## 🎯 **Problema**
Entiendo que el MCP funciona, pero no veo casos de uso prácticos para mi flujo de trabajo diario.

## ✅ **Respuesta y Solución**

### **🚀 Casos de Uso Reales**

#### **1. 🔍 Debugging y Resolución de Problemas**

**Escenario**: Tu servidor web Go tiene problemas de rendimiento.

**Workflow con MCP**:
```
1. Usa get-context con query="performance issues HTTP handler gin middleware"
2. [Claude obtiene contexto relevante de tu proyecto]
3. Usa remember-conversation con key="performance-fix-2025" content="El problema era el middleware de logging que bloqueaba el pipeline. Solución: usar goroutines para logging asíncrono" tags=["performance", "debugging", "middleware"]
```

**Resultado**: Claude entiende tu arquitectura y recuerda la solución para futuras consultas.

---

#### **2. 🏗️ Desarrollo de Nuevas Funcionalidades**

**Escenario**: Necesitas añadir autenticación JWT a tu API.

**Workflow con MCP**:
```
1. Usa analyze-project para entender la estructura actual
2. Usa fetch-docs con library="golang-jwt" topic="middleware implementation"
3. Usa get-context con query="HTTP middleware authentication gin framework"
4. [Implementas la funcionalidad]
5. Usa remember-conversation con key="jwt-auth-implementation" content="Añadimos JWT auth usando gin middleware. Config en config/auth.go, middleware en handlers/auth.go. Secret key desde ENV variable JWT_SECRET" tags=["auth", "jwt", "security", "implementation"]
```

**Resultado**: Documentación automática de decisiones y implementación.

---

#### **3. 📋 Code Review y Mejores Prácticas**

**Escenario**: Revisión de código antes de merge.

**Workflow con MCP**:
```
1. Usa get-context con query="security best practices HTTP server golang"
2. Usa dependency-analysis con includeTransitive=true para revisar dependencias
3. [Claude sugiere mejoras basadas en el contexto de tu proyecto]
4. Usa remember-conversation con key="security-checklist-2025" content="Checklist de seguridad: validar inputs, rate limiting, HTTPS only, JWT expiration, sanitizar logs" tags=["security", "checklist", "review"]
```

### **💡 Ventajas Reales vs Flujo Tradicional**

| Aspecto | Sin MCP | Con MCP |
|---------|---------|---------|
| **Contexto** | Se pierde entre sesiones | Persistente y acumulativo |
| **Conocimiento del proyecto** | Hay que reexplicar | Claude "conoce" tu proyecto |
| **Decisiones** | Se olvidan | Documentadas automáticamente |
| **Debugging** | Empezar desde cero | Contexto histórico disponible |
| **Onboarding** | Documentación manual | Contexto consultable |
| **Code Review** | Sin contexto histórico | Con historia de decisiones |

### **📈 ROI (Return on Investment)**

**Inversión inicial**: 2-3 horas configurando y aprendiendo
**Ahorro semanal**: 3-5 horas no reexplicando contexto
**Beneficio adicional**: Documentación automática de decisiones

### **🎯 Empezar Gradualmente**

**Semana 1**: Solo usa `analyze-project` y `get-context`
**Semana 2**: Añade `remember-conversation` para decisiones importantes
**Semana 3**: Incorpora `fetch-docs` para referencias
**Semana 4**: Usa `dependency-analysis` para mantenimiento

### **📚 Recursos Adicionales**
- Ver archivo completo [FAQ-06-use-cases.md](../../docs/faq/FAQ-06-use-cases.md)
- Ver [MANUAL.md](../../MANUAL.md) para workflows detallados

---

**¿Alguno de estos casos de uso aplica a tu trabajo?** ¿Cuál te parece más útil para empezar?
