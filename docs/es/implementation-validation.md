# Validación ejecutable de la implementación

Comprobado: 2026-09-30.

Registra evidencia de runners Ubuntu limpios de GitHub. Son máquinas desechables, por lo que demuestran comportamiento reproducible sin tocar producción.

## Evidencia

### Write/read real MCP → Gateway → Broker → Linux

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283767952

Ubuntu 24.04, Gateway non-root, Broker root con socket protegido, MCP Streamable HTTP local, write/read reales, verificación externa del archivo/SHA-256 y cadena audit válida con 2 eventos.

~~~text
MCP VPS Agent Gateway passou pela VPS efemera do GitHub.
sha256:
572890f111b50050dd1c405e724a0cd2b31dee2df32bd52c30d3db6b1853eea1
~~~

### Prueba negativa /etc/shadow

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283831850

`file.read_test` intenta `/etc/shadow`. Solo pasa con denial/error y audit válido. **PASS**.

### Restart systemd real

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36283884675

~~~text
service.status
service.restart
service.status
PID antes: 3124
PID después: 3173
ActiveState: active
~~~

Audit con 3 eventos válido. **PASS**.

## Aceptación E2E Docker

Workflow:
https://github.com/josemirmoura/mcp-vps-agent-gateway/actions/runs/36324349224

**PASS** en Ubuntu 24.04 limpio con Dockerfile + compose.yaml reales.

Prueba health Broker/Gateway; CRUD completo Scoped; denial de `/etc/shadow`; `shell.exec` dentro de raíz y sin acceso a shadow; systemd host; Docker/Compose host; diagnósticos; audit chain.

Broker usa namespaces host para operaciones nativas. Shell Scoped usa mount namespace systemd con root vacío read-only y binds solo de runtime/toolchain y roots autorizadas.

Docker es packaging, no authorization boundary. Broker privilegiado es código trusted host-root; policy server-side es autoridad efectiva.

## CI

Valida `go vet`, `go test -race`, builds amd64/arm64, simulación adversarial, Docker smoke, inspect sanitizado, systemd hardening, transient jobs, govulncheck/gosec/gitleaks, fuzzing bounded, límites de body/concurrencia, SO_PEERCRED negativo, hardlink/symlink/rename-race, Trivy y CycloneDX SBOM.

El commit más reciente debe estar green antes de merge.

## Frontera de privilegio

~~~text
/run/vps-agent              root:vps-agent 0750
/run/vps-agent/broker.sock  root:vps-agent 0660
/var/lib/vps-agent          root:vps-agent 0750
~~~

Policy, secretos Broker y SQLite son root-only. La CI falla si Gateway puede leer broker.env, policy.yaml o state.db.

## ChatGPT Web + OAuth real

El 2026-09-28 se validó:

~~~text
ChatGPT Web
 -> HTTPS MCP
 -> OAuth/OIDC discovery
 -> DCR + PKCE
 -> operador dedicado
 -> Gateway token validation
 -> Broker subject + policy
 -> system.info
 -> audit
~~~

El flujo alcanzó `CHATGPT WEB CONNECTION VERIFIED` / `INSTALLATION COMPLETE`. Valores específicos del entorno no se publican.

Esto cierra Gate 0A integrado. RC5 añade discovery-only del techo, secret enforcement anidado, elicitation MCP nativa, safety annotations, abuse limits y race tests. Aún se requiere aceptación clean-install final antes de `v0.1.0`.

## Qué prueba

~~~text
MCP client
 -> Streamable HTTP
 -> non-root Gateway
 -> protected Unix socket
 -> privileged Broker
 -> server-side policy
 -> real Linux operation
 -> Broker audit
~~~

## Qué no prueba

No prueba fiabilidad prolongada de producción, matriz formal amplia, remote audit anchoring, Full/admin shell en producción ni recovery de largo plazo. Full/admin existen pero están deshabilitados por defecto.

## Prueba interactiva reproducible

`proof/request.json` soporta `file_write_read`, `file_read_denied`, `service_restart`. Cambiarlo en PR dispara `.github/workflows/ephemeral-proof.yml` y produce evidencia, audit status, journals, permisos, estado systemd y contenido/hash cuando aplique.
