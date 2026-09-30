# Arquitectura

## Objetivo

Dar a un cliente de IA acceso operativo útil a una VPS Linux manteniendo a la VPS, y no al modelo, como autoridad de seguridad.

> **El LLM nunca es la frontera de seguridad.**

## Runtime canónico

Dos procesos del proyecto, ambos en Go:

~~~text
ChatGPT / cliente MCP
        |
        | MCP Streamable HTTP
        v
+-----------------------------+
| vps-agent-gateway           |
| sin privilegios             |
| MCP + auth + schemas        |
| tools canónicas             |
+-------------+---------------+
              |
              | Unix Domain Socket
              v
+-------------+---------------+
| vps-agent-broker            |
| privilegiado, local-only    |
| policy autoritativa         |
| SQLite + locks + jobs       |
| secretos + auditoría        |
| files + systemd + Docker    |
| ejecución sandboxed         |
+-------------+---------------+
              |
              v
     Linux / systemd / Docker
~~~

Proxy inverso o túnel privado soportado es infraestructura de ingreso, no un tercer servicio.

## Go

El SDK Go oficial de MCP es Tier 1 y soporta MCP 2026-07-28. Una sola lengua reduce packaging, dependencias y mantenimiento sin eliminar la frontera de privilegio.

## Gateway

Corre sin root. Expone MCP, valida OAuth/OIDC y schemas, normaliza tools/resources, hace preflight no autoritativo y llama al Broker por socket Unix.

Nunca debe ejecutar como root, recibir Docker socket, abrir SQLite privilegiado, leer secretos plaintext, convertirse en autoridad ni aprobar su propia elevación.

## Broker

Es la frontera privilegiada. Cada llamada se reautoriza contra:

~~~text
subject
+ tool canónica
+ resource canónico
+ action
+ policy actual
+ grant/lease/job cuando aplique
~~~

Posee evaluación de policy, SQLite, idempotencia, locks, jobs, filesystem seguro, Docker/systemd, sandbox, secretos y audit. Gateway es un deputy no confiable.

## Modelo de capacidad

Un catálogo amplio, autoridad por policy. El operador puede autorizar una raíz, varias o el host entero.

Techo físico y raíces lógicas son distintos. El techo es límite máximo, no permiso. Standard permite discovery-only de nombres de directorios inmediatos. Delegaciones dinámicas están ligadas a subject, perfil `read`/`work`/`compose` y TTL opcional.

Crear request no autoriza. Con MCP elicitation, el cliente muestra confirmación nativa. Broker valida request, subject y token one-time. Las tools de confirmación no son visibles al modelo. Sin elicitation se falla cerrado hacia fallback del operador. Revocación actúa desde state sin restart.

Archivos secretos forman una frontera anidada: `.env` necesita grant temporal exact-path. Templates comunes siguen siendo contenido normal. El sandbox de shell también oculta paths protegidos.

Filesystem es un eje; systemd, Docker, shell, red y administración se controlan aparte. Ver [product-model.md](product-model.md).

## Policy y approvals

Policy Engine vive dentro del Broker. Policies deny-by-default. Presets: Controlled, Scoped y Full. Full está off por defecto.

Routine Scoped no debe pedir humano. Elevación temporal usa un flujo separado; MCP elicitation es la vía preferida para grants normales, con Broker como autoridad, subject-bound, nonce one-time y anti-replay. Ruta admin separada queda para fallback/MFA.

## Full

Expande capacidades explícitas:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

No implica red irrestricta. Es temporal, revocable y gated.

## Transporte

MCP Streamable HTTP sobre HTTPS estable. Core stateless; no crear WebSocket/sesión propia. Jobs y leases usan handles explícitos y el SDK oficial maneja negociación.

## Filesystem

Nunca autorizar por prefix string. Preferir `openat2` restrictivo; fallback con directory-FD walk seguro u optar por fail closed. Nunca degradar a checks string.

## Process isolation

Transient systemd units, cgroups, NoNewPrivileges, PrivateTmp, restricciones FS, MemoryMax, TasksMax, deadline, output limit y cancellation. Landlock es defensa extra.

## Docker

Gateway nunca recibe `/var/run/docker.sock`. Docker se implementa como operaciones tipadas del Broker.

## Jobs

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Modelo interno durable y autoritativo, independiente de la conexión HTTP. MCP Tasks puede mapearse después.

## State

SQLite solo Broker, transacciones cortas. Puede guardar approvals, leases, jobs, idempotencia, locks y audit metadata.

Locks usan `resource`, `owner/action_id`, `fencing_token` monotónico y `expires_at` para evitar ABA.

Idempotencia usa journal:

~~~text
PENDING -> external effect -> DONE
~~~

Se persiste PENDING antes del efecto; tras crash se reconcilia en vez de repetir ciegamente.

## Secretos

Preferir archivos root-owned y systemd credentials. Broker resuelve referencias e inyecta al proceso objetivo. Gateway/tools no recuperan plaintext.

## Auditoría

Gate 1: audit local estructurado.
Scoped production: integridad/secuencia durable.
Antes de Full: hash chain tamper-evident, checkpoint remoto y recovery probado.

## Downstream

Upstreams allowlisted, tools namespaced, cambios fingerprinted/revisados, resultados como datos no confiables. Resultados nunca cambian policy, crean leases, registran servers ni exponen secretos.

## No objetivos iniciales

No es gateway universal, multi-tenant control plane, scheduler distribuido, root-shell genérico, Kubernetes, reemplazo de SSH ni soporte de todo cliente MCP.

> Objetivo inicial: demostrar de forma segura una operación pequeña, valiosa y auditable en una VPS real.
