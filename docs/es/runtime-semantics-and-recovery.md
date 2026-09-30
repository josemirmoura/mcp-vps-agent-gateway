# Semántica runtime y recovery

## 1. Expiración durante jobs

Job elevado deriva grant inmutable al inicio.

~~~text
job_deadline <= grant_expiry
~~~

Al expirar: bloquear nuevas mutaciones, terminar graciosamente, forzar tras grace period bounded, persistir estado y auditar. Jobs Scoped normales no necesitan grant. Si un job admin debe durar más, requiere completion grant específico.

## 2. Concurrencia SQLite

SQLite single-writer es aceptable inicialmente:

- solo Broker abre DB;
- transacciones cortas;
- nunca durante shell/deploy/migration/Docker;
- persistir transición, commit, trabajo externo, siguiente transición;
- locks lógicos con deadlines;
- WAL;
- busy timeout/backoff.

Migrar solo si contención medida importa.

### Fencing

Locks con TTL llevan token monotónico. Job stale no puede liberar/commit con token antiguo. Release compara resource + owner/action_id + token.

~~~text
A token 1
A expira
B token 2
A vuelve
A NO puede sobrescribir B
~~~

### Crash de idempotencia

~~~text
PENDING -> effect -> DONE
~~~

Journal antes del side effect. Crash después del efecto no auto-repite; reconciliación o resultado indeterminate.

## 3. Idempotencia como metadata

Gateway/integration genera retry identity y la liga a subject + canonical tool + normalized request hash + invocation identity. Sin identidad estable, no auto-retry state changes. Mismos argumentos no implican retry.

## 4. Secretos

Sin tools de lectura plaintext. Preferir archivos root-owned fuera del repo y systemd credentials. Broker resuelve `secret_ref` e inyecta al target. Gateway no lee, tools no enumeran, audit no contiene secrets y se redige salida cuando sea práctico.

## 5. Fricción de aprobación

Trabajo rutinario usa Scoped preautorizado; aprobación out-of-band queda para elevación excepcional.

## 6. Recovery

Broker restart: reconciliar state con units/procesos, adoptar o terminar según policy.

SQLite corrupto: fail closed, parar writes, invalidar approvals, revocar grants, preservar DB, restaurar backup, reconciliar jobs y reactivar tras integridad.

Grant erróneo:

~~~text
vps-agent revoke-all
~~~

Revoca grants, bloquea nuevos starts elevados, termina/cuarentena jobs según emergency policy y audita. Rotar credenciales si se sospecha compromiso.

Quitar la ruta Gateway no debe detener workloads.

## 7. Backups

Respaldar policy, SQLite, audit checkpoints y unit files. No incluir plaintext secrets salvo cifrado intencional.

## 8. Readiness

Scoped: gates iniciales, comportamiento real probado, recovery, secret path, retiro del control plane sin afectar workloads, límites de soporte documentados y fiabilidad acorde al claim.

RC cumple gates de implementación/E2E pero no historial R4 prolongado.

Full: además expiry semantics, revoke-all, approval flow, remote audit checkpoint y network elevation separation.
