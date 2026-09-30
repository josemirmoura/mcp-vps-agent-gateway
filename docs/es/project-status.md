# Estado y madurez del proyecto

## Etapa actual

**PRE-RELEASE / PRODUCTIZACIÓN / GATE E2E CHATGPT COMPLETO**

La implementación Go Docker-first tiene validación repetible en runners limpios para runtime Scoped, lifecycle y operaciones host.

Todavía no hay release estable, promesa de compatibilidad de producción, historial prolongado en VPS real ni claim de producción para Full/admin shell.

La evidencia de laboratorio cubre CRUD de filesystem, denegaciones, shell/jobs sandboxed, systemd host, Docker/Compose host, diagnóstico, lifecycle y auditoría. El 2026-09-28 el camino soportado OAuth self-hosted se ejercitó contra ChatGPT Web en una VPS real mediante `system.info` autenticado observado por Broker y registrado en audit. Esto cierra Gate 0A sin crear claim estable.

## Escalera de madurez

### R0 — Arquitectura

Documentación sin implementación ejecutable.

### R1 — Ruta producto + Gate 0B

- Gate 0A registra la ruta real. **Completado 2026-09-28 para OAuth integrado + ChatGPT Web.**
- MCP Inspector pasa.
- POC segura read/write solo en raíz descartable.

### R2 — Piloto privilegiado tipado

Gateway/Broker separados, un servicio no crítico manejable por tools tipadas, denegación fail-closed y auditoría local.

### R3 — Piloto Scoped

Un stack real bajo policy explícita, comportamiento duradero por días, recovery/denial ejercitados y sin Full rutinario.

### R4 — Producción Scoped endurecida

Auth validada, state duradero, jobs/retries/locks, secrets, backup/recovery y monitoring probados. Solo entonces puede llamarse production-capable para Scoped.

### R5 — Elevated/Full

Además de R4: Full explícito, aprobación out-of-band, grants temporales, revoke-all, network elevation separada, audit tamper-evident con checkpoint remoto y recovery admin probado.

## Full por defecto

~~~yaml
features:
  full_mode_enabled: false
~~~

Full no es requisito del éxito. Un Scoped sólido es un endpoint de producción válido.

## Evidencia sobre popularidad

Stars/forks no son evidencia de producción. Prioriza builds reproducibles, tests, releases, deployment history, recovery tests, incident learnings, mantenimiento, security review y revisión externa.

## Próximo hito

El hito actual es **aceptación clean-install del propietario de `v0.1.0-rc.5`**. RC5 incluye MCP elicitation nativa, inventario discovery-only del techo, enforcement de secretos anidados y safety annotations.

El bloqueador para `v0.1.0` estable es instalar el tag exacto RC5 y probar OAuth real, `system.info` auditado, UX nativa, discovery sin contenido, operaciones Scoped, denegación/excepción/revocación de `.env`, revocación root y lifecycle. La fiabilidad de larga duración queda después. Ver [operator-acceptance.md](operator-acceptance.md) y [implementation-validation.md](implementation-validation.md).
