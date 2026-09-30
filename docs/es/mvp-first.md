# Trilha MVP-first

La arquitectura completa es el norte, no el primer hito.

## Gate -1 — Adoptar, adaptar o construir

Evalúa productos y servidores MCP existentes. Adopta si cumplen, adapta si están cerca y construye solo si falta la combinación requerida.

Objetivo diferencial:

~~~text
ChatGPT Web
+ VPS autocontrolada
+ policy server-side
+ autonomía Scoped
+ elevación temporal opcional
+ Broker bajo control del operador
~~~

## Gate 0A — Probar superficie ChatGPT

**Completado 2026-09-28 para OAuth integrado + ChatGPT Web.**

Antes del camino privilegiado se debe validar el producto cliente real. No supongas que un MCP privado con escritura puede conectarse a cualquier plan.

Rutas:

1. workspace/plano con private full MCP write;
2. app/plugin elegible con remote write;
3. MCP Inspector mientras la distribución no esté resuelta.

Registrar:

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

Si una cuenta futura no ofrece custom MCP, cambia la distribución o detente en ese boundary. No debilites el servidor.

## Gate 0B — POC seguro

Go + SDK oficial, usuario sin privilegios. Solo:

~~~text
system.info
file.read_test
file.write_test
~~~

Filesystem restringido a `/tmp/vps-agent-poc/`. Sin root, Docker, systemd write, SQLite, secretos, Full, approvals o shell genérico.

Éxito: Inspector pasa, tools descubiertas, read/write según producto, paths prohibidos fallan, errores claros y retries seguros.

## Gate 1 — Una acción privilegiada tipada

Gateway non-root + Broker privilegiado por Unix socket. Añade `service.status` y `service.restart` para una unidad no crítica. Mantén fuera admin shell, Full, approval UI, Docker amplio y audit remoto.

## Gate 2 — Piloto Scoped real

Policy explícita para un stack. Añade solo lo necesario:

~~~text
system.info
file.read
service.status
service.restart
docker.logs
docker.action(restart)
job.status
~~~

Mide frecuencia, éxito/fallo, correcciones, false denials, capacidades faltantes y recovery. Pasa cuando Scoped resuelve trabajo útil sin elevación rutinaria.

## Gate 3 — Durabilidad

Cuando haga falta: SQLite del Broker, jobs duraderos, idempotency identity, locks, systemd credentials/secret refs y audit estructurado más fuerte.

## Gate 4 — Writes amplios

Según necesidad: `file.write`/`file.patch`, updates validados, acciones Docker/systemd tipadas y `shell.exec` sandboxed en raíces Scoped. Shell genérico no es requisito.

## Gate 5 — Elevación temporal

Solo si Scoped es insuficiente: requests de elevación, aprobación humana out-of-band, leases temporales, revoke-all, anchoring remoto y flag Full. Solo después considerar `shell.exec_admin`.

## Regla

Cada componente necesita evidencia del gate anterior.

> ¿Puede el cliente real completar trabajo valioso, seguro y fiable en un servidor real?
