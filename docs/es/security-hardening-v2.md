# Hardening de seguridad

## 1. Autorización en Broker

Gateway no autoriza trabajo privilegiado. Cada operación une subject + tool canónica + resource + action + policy + grant cuando aplica. No hay ambient authority heredada de pasos previos.

## 2. Expansión de permisos

Routine work usa Scoped. MCP puede pedir una nueva raíz dentro del techo; el request no concede nada. Broker guarda pending approval tipado con subject/root/profile/lifetime. Solo una frontera humana separada activa.

Delegaciones dinámicas son state del Broker: no editan `policy.yaml`, no amplían techo, no habilitan acciones ausentes. Revocación puede ser in-band porque reduce autoridad; expiry se aplica server-side y shell jobs temporales no pueden durar más.

Annotations/host confirmations son UX, no security boundary.

Con MCP elicitation, la UI nativa confirma root o protected file. El modelo no recibe self-approval tool ni approval token. Broker liga decisión a request, subject, resource, profile y expiry. Sin elicitation, queda pending para fallback.

### Elevación admin

MCP pide, nunca aprueba. Humano usa flujo autenticado separado, step-up/MFA preferible, nonce one-time, rate limits/cooldowns, URL expirante y sin decisiones en GET. Full solo tras R5.

## 3. Full capability-based

`shell.admin`, `filesystem.read:any`, `filesystem.write:any`, `docker.admin`, `systemd.admin`. `network.unrestricted` siempre separado.

## 4. Replay safety

Replay-safe requiere idempotency identity de infraestructura; misma identidad/request devuelve resultado guardado y request diferente conflict. Non-replay-safe nunca retry ciego; usa action IDs, locks/state y autorización adecuada. LLM no inventa key.

## 5. SQLite

Solo Broker abre DB privilegiado. APIs estrechas para Gateway/aprobación. Transacciones cortas, sin comandos externos dentro.

## 6. Filesystem

Sin string prefixes. Preferir `openat2`; fallback directory-FD seguro. Si no es seguro, privileged writes fail closed.

## 7. Kernel

Health reporta capabilities. Sin Landlock mantener systemd/cgroups y reportar degradación. Sin `openat2`, fallback documentado o fail closed. Nunca fallback inseguro silencioso.

## 8. Tool/result trust

Outputs, logs, web y downstream MCP son untrusted. Nunca mutan policy, crean/extienden grants, cambian trust tier, registran MCP, exponen secrets ni bypass network. Downstreams configurados out-of-band y cambios fingerprinted/reviewed.

## 9. Secretos

Root-owned files y systemd credentials. Gateway no lee plaintext. Sin tool `secret.read_plaintext`. Redactar credenciales cuando sea práctico.

## 10. Audit por madurez

R1/R2 local estructurado. R3/R4 ordering/integrity durable y retención. R5/Full sequence monotónica, hash chain, checkpoint/signature, remote anchoring y detección de gaps. No llamar tamper-proof a logs locales root.

## 11. Negative tests obligatorios

Traversal/symlink escape, recursos no autorizados, subject incorrecto, grants stale/revocados, idempotent retry once, no blind retry, malicious tool result sin grant, downstream URL arbitraria, Gateway sin Docker socket/SQLite, kernel feature missing sin fallback inseguro, stale lock owner bloqueado y crash post-effect sin duplicar.

Antes de R5: elevation spam rate-limited, self-approval imposible, Full sin network no tiene egress irrestricto, tampering detectable y revoke-all funciona.
