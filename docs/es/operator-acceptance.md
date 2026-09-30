# Gate final de aceptación del operador

Lo opera deliberadamente el propietario. Es la última etapa antes de congelar `v0.1.0`.

No ejecutarlo casualmente sobre una instalación importante. Preferir VPS limpia o entorno elegido para pruebas de lifecycle.

## Condiciones

- tag RC exacto;
- Linux clase Ubuntu 24.04;
- Docker Engine 24+ + Compose v2;
- mínimo 2 GB RAM para ZITADEL;
- control de DNS;
- TCP 80/443;
- ChatGPT Web con Developer Mode/custom MCP realmente disponible;
- para writes, confirmar que el producto actual expone esas acciones.

Registrar tag/commit, OS/arquitectura, perfil, hostname y timestamps, nunca secretos.

## 1. Instalación limpia

~~~bash
git clone --branch v0.1.0-rc.5 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

Confirmar banner/versión, requisitos, perfiles comprensibles, autoridad antes de startup, Standard con `/opt` como techo y sin proyecto autorizado, posibilidad de otro techo, explicación de que techo no es grant, sin escalada silenciosa y sin pedir credenciales VPS/SSH.

## 2. Seguridad pública/OAuth

DNS correcto, HTTPS válido, protected-resource metadata, OIDC discovery, DCR + PKCE, login dedicado, subject estable y MCP sin auth fail-closed.

## 3. ChatGPT E2E

Crear app con `/mcp`, completar OAuth, invocar `system.info`, volver al terminal y Enter. No imprimir `INSTALLATION COMPLETE` hasta observar la llamada por Gateway → Broker → policy → execution → audit.

## 4. UX nativa, discovery y operaciones

Antes del grant, `permissions.discover_scope` debe devolver solo nombres inmediatos sin contenido.

Solicitar raíz desechable:

- UI nativa MCP elicitation, no iframe antiguo;
- path, perfil, duración y techo visibles;
- exact-ceiling muestra advertencia fuerte;
- aprobar raíz estrecha con `read`/`work`/`compose`;
- revocar y confirmar denial inmediato.

Dentro: read/write archivo, service/Docker permitido, shell/job bounded si enabled y denial fuera de scope.

### Secretos

- `.env.example` legible;
- `.env` denegado por read/hash/shell aun con padre autorizado;
- solicitar acceso temporal exacto;
- UI nativa muestra archivo/perfil/expiración;
- aprobar, probar, revocar;
- inaccesible de nuevo;
- contenido secreto ausente de logs/audit.

No ampliar policy para hacer pasar tests.

## 5. Observabilidad

~~~bash
bash scripts/diagnose.sh status
bash scripts/diagnose.sh health
bash scripts/diagnose.sh logs 100
bash scripts/diagnose.sh audit 100
bash scripts/diagnose.sh bundle
~~~

Confirmar salud/logs útiles, audit independiente, cadena válida, bundle restrictivo y sin secretos.

## 6. Update/rollback

~~~bash
bash scripts/version.sh --check
VPS_AGENT_UPDATE_REF=<next-rc-tag> bash scripts/update.sh
~~~

Confirmar versiones, resumen, backup, persistencia policy/state/identity, migración, verificación y rollback automático en prueba controlada.

## 7. Safe remove/reinstall

`bash scripts/remove.sh safe`: runtime desaparece, grants revocados, credenciales rotadas, estado/config/identity preservados y workloads externos intactos. Reinstalar y retomar.

## 8. Purge

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

Eliminar solo artefactos MCP y preservar recursos administrados.

## Criterio

Pasa solo si todo aplicable funciona sin debilitar seguridad. Si falla: registrar sin secretos, corregir, rerun tests, repetir sección y solo luego promover `v0.1.0`. Full/R5 y fiabilidad R4 prolongada son aparte.
