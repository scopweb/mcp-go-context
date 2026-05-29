---
title: Buenas Prácticas de Memoria
description: Cómo sacar el máximo partido al sistema de memoria persistente sin generar ruido.
---

Esta guía reúne las mejores prácticas que hemos ido descubriendo al usar `mcp-go-context` en proyectos reales. El objetivo es mantener una memoria de **alta calidad**, útil a largo plazo y sin ruido innecesario.

## 1. Cuándo guardar una decisión (y cuándo no)

### Guarda cuando...

- La decisión afecta la arquitectura o el futuro del proyecto.
- Es un fix no obvio que podría repetirse.
- Estableces una convención o patrón que el equipo debe seguir.
- Hay alternativas reales que se consideraron y vale la pena recordar por qué se descartaron.

### No guardes cuando...

- Es contexto temporal de la sesión actual.
- Es algo que se puede deducir fácilmente mirando el código.
- Es una decisión que probablemente cambiarás en las próximas semanas.
- Es una nota personal o recordatorio de "hacer esto después".

**Regla de oro**: Si dentro de 3 meses alguien nuevo (o tú mismo) pregunta “¿por qué hicimos esto?”, probablemente merece ser guardado.

## 2. Cómo escribir un buen `save-decision`

Un buen registro de decisión tiene tres partes clave:

### Ejemplo bueno

```text
save-decision
  key="auth-architecture-2026"
  decisionType="architecture"
  content="Usaremos JWT + refresh tokens rotativos en lugar de sesiones en base de datos"
  reason="Necesitamos escalabilidad horizontal y soporte para clientes móviles sin mantener estado en el servidor. Las sesiones en BD complicaban el scaling y generaban problemas de consistencia."
  alternatives=["Sesiones en Redis", "OAuth2 + DB sessions", "Magic links sin tokens"]
  tags=["auth", "scalability", "mobile"]
```

### Por qué funciona bien

- Tiene un `key` descriptivo y con fecha (fácil de encontrar después).
- El `reason` explica el **porqué** real (no solo qué se hizo).
- Incluye alternativas concretas que se evaluaron.
- Los tags permiten filtrar después por tema.

**Consejo**: Cuanto más explícito seas en el `reason`, más útil será la memoria cuando `suggest-promotions` la evalúe más adelante.

## 3. El flujo correcto de promoción (2026)

El flujo más efectivo actualmente es:

1. Durante el trabajo → `save-decision` (con buena calidad).
2. Al terminar una sesión importante o antes de cambiar de contexto → `suggest-promotions`.
3. Revisas las sugerencias (normalmente 3-6).
4. Solo promueves las que realmente tienen valor a largo plazo con `promote-memory`.

**No** uses `promote-memory` directamente sin pasar por `suggest-promotions`. La herramienta de sugerencias es tu filtro de calidad.

## 4. Cómo usar `suggest-promotions` de forma efectiva

- Úsala al final de sesiones largas o complejas.
- Úsala también cuando sientas que "tienes muchas memorias y no sé qué es importante".
- No promociones todo lo que te sugiere. Su trabajo es darte opciones; el tuyo es decidir.
- Después de promover, puedes volver a ejecutarla para ver si aparecen nuevas candidatas.

## 5. Organización y tags

Usa tags de forma consistente. Algunos patrones que funcionan bien:

- Temáticos: `auth`, `database`, `performance`, `security`, `deployment`
- Tipo de decisión: `architecture`, `fix`, `convention`, `tech-debt`
- Contexto: `mobile`, `scalability`, `legacy`

Evita tags demasiado genéricos como `important` o `todo`. Son poco útiles cuando buscas después.

## 6. Uso del Dashboard

El Dashboard es especialmente útil para:

- Revisar periódicamente (cada 2-4 semanas) qué memorias se han acumulado.
- Ver de un vistazo cuántas memorias tienes promovidas.
- Hacer limpieza: a veces una memoria que parecía importante hace meses ya no lo es.
- Promover varias cosas seguidas sin tener que escribir comandos.

Muchos usuarios descubren que el Dashboard es donde realmente hacen la mayor parte del trabajo de "gestión de memoria".

## 7. Anti-patrones comunes

| Anti-patrón                        | Por qué es malo                              | Mejor alternativa |
|------------------------------------|----------------------------------------------|-------------------|
| Promover todo lo que guardas       | La memoria se llena de ruido                 | Usar siempre `suggest-promotions` como filtro |
| Guardar decisiones sin `reason`    | Después nadie recuerda por qué se tomó       | Exigirte escribir al menos 1-2 frases de razón |
| Usar keys genéricos ("decision-1") | Es imposible encontrarlas después            | Usar nombres descriptivos + fecha si hace falta |
| No revisar nunca las memorias      | Se acumula basura que nadie usa              | Revisión ligera cada 3-4 semanas vía Dashboard |
| Promover notas temporales          | Contamina la memoria de alto valor           | Dejarlas solo en SessionMemory |

## 8. Consejos avanzados

- Combina `save-decision` con buen `reason` + tags → las sugerencias de `suggest-promotions` mejoran notablemente.
- Usa el Dashboard como herramienta de revisión, no solo como visualizador.
- No tengas miedo de **no** promover algo. Es mejor tener pocas memorias de alta calidad que muchas de calidad media.
- Si una memoria ya está claramente documentada en el código o en la arquitectura, probablemente no necesita estar también en memoria persistente.

---

Siguiendo estas prácticas, la mayoría de usuarios consiguen mantener una memoria persistente útil, pequeña y de alto valor durante meses o años.