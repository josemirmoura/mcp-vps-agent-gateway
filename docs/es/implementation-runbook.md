# Runbook de implementación

> Ruta histórica de construcción. El runtime Scoped amplio, state/jobs durables, OAuth integrado y código opcional de elevación ya existen. Usa este documento para entender la secuencia, no como estado actual. El estado vive en [project-status.md](project-status.md).

La arquitectura canónica está en `docs/architecture.md` y los gates en `docs/mvp-first.md`.

## 0. Reglas

1. leer `docs/README.md`;
2. completar Gate -1;
3. completar Gate 0A;
4. no construir más allá del gate actual;
5. usar Go para ambos binarios;
6. mantener Gateway sin privilegios y Broker local-only.

## 1. Layout

~~~text
cmd/
  vps-agent-gateway/
  vps-agent-broker/
internal/
  gateway/
  broker/
  shared/
policy/
deploy/
tests/
~~~

Runtime:

~~~text
/etc/vps-agent/       configuración/metadatos de secretos
/run/vps-agent/       broker.sock
/var/lib/vps-agent/   state/jobs/audit
~~~

## 2. Gate 0B

Construye solo Gateway con SDK Go oficial y Streamable HTTP. Tools: `system.info`, `file.read_test`, `file.write_test`. Root de prueba: `/tmp/vps-agent-poc/`.

Requisitos: usuario non-root, sin Docker socket/SQLite/shell genérico, schemas, output limits, path confinement y MCP Inspector. Usa una interfaz Executor interna para sustituir luego el executor local por BrokerClient.

## 3. Gate 1 — Broker

Añade Broker con Unix Domain Socket:

~~~text
/run/vps-agent/       root:vps-agent 0750
/run/vps-agent/broker.sock root:vps-agent 0660
~~~

Comprueba peer credentials cuando sea práctico. Operaciones iniciales: ping, system.info, service.status, service.restart para una unidad no crítica explícita. Autorización deny-by-default.

Auditoría: action ID, subject, tool/resource canónicos, decisión, resultado/exit y duración. Nunca secretos brutos.

## 4. Gate 2 — Scoped

Policy parsing/validation dentro del Broker. Unknown fields/capabilities fallan cerrados. Define un stack real y añade solo file.read, service status/restart, docker logs/action, job.status según necesidad. Docker sigue Broker-only y tipado.

## 5. Gate 3 — State durable

Solo Broker abre `/var/lib/vps-agent/state.db`. WAL, transacciones cortas, busy timeout/backoff, locks lógicos con deadline y fencing token.

State: jobs, operation journal/idempotencia, resource locks, audit metadata y después approvals/grants.

Antes de side effect:

1. claim `PENDING`;
2. commit;
3. efecto externo;
4. `DONE`.

Crash entre efecto y DONE entra en reconciliación; nunca repite a ciegas.

Idempotencia liga subject + tool + request hash + invocation identity. Argumentos iguales no significan retry.

## 6. Jobs

`job.start/status/tail/cancel`. Job en unit/cgroup gestionado cuando convenga. Persistir owner, policy/grant, resource/action, unit/pid, deadline, status y exit code. Perder conexión MCP no pierde job.

## 7. Secretos

Preferir archivos root-owned fuera del repo y systemd credentials. Broker resuelve referencias. Sin tool de plaintext secret. Testear stdout/stderr/audit contra fugas.

## 8. Gate 4 — Writes

Resolución segura y writes atómicos. Preferir `openat2`; fallback seguro o fail closed.

~~~text
backup/write temporal
 -> validador nativo
 -> replace atómico
 -> reload/restart
 -> healthcheck
 -> rollback
~~~

`shell.exec` solo si typed tools no bastan, con transient unit/cgroup, cwd roots, timeout, MemoryMax, TasksMax, output limits, cancellation, environment filtrado y network policy.

## 9. Auth/ChatGPT

El camino integrado self-hosted OAuth está implementado. Sigue [chatgpt-integration.md](chatgpt-integration.md) y [authentication.md](authentication.md). Gateway valida token; Broker revalida subject/policy.

## 10. Gate 5 — Elevación

Existe machinery opcional acceptance-tested, sin claim Full/R5. Full sigue off y Scoped es el camino soportado.

Si se usa: request, aprobación humana out-of-band, nonce one-time, capabilities temporales, expiry, revoke-all, network elevation separada y audit reforzado. Solo después considerar `shell.exec_admin`.

## 11. Grants/jobs

~~~text
job_deadline <= grant_expiry
~~~

Al expirar: bloquear nuevas mutaciones, terminar graciosamente, forzar tras grace bounded, persistir estado final y auditar. Completion grant especial requiere aprobación explícita.

## 12. Recovery

Broker restart: reconciliar jobs persistidos con units/procesos.

SQLite corrupto: fail closed, invalidar approvals, revocar grants, preservar DB, restaurar backup, reconciliar jobs y reactivar solo tras checks.

Emergencia:

~~~text
vps-agent revoke-all
~~~

Fuera de la superficie IA; revoca grants y bloquea nuevos starts.

## 13. Hardening systemd

Gateway: User=vps-agent, NoNewPrivileges, ProtectSystem=strict, ProtectHome, PrivateTmp, ReadWritePaths mínimos, sin Docker socket.

Broker: root-owned, sin TCP, Unix socket, API mínima, paths fijos, journald y restart-on-failure. Revisar con `systemd-analyze security`.

## 14. Aceptación de producción

Scoped exige gates, negative auth tests, filesystem safe, recovery, secrets, camino de cliente y workloads sobreviviendo a retirar Gateway/Broker.

Full añade approval, expiry, revoke-all, separación de network, remote audit checkpoint y admin shell containment/recovery.

## 15. Rollback

Gateway/Broker debe ser control plane opcional. Pararlo o quitar su ruta no puede detener workloads existentes.
