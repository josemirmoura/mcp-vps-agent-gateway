# Flujo de primera ejecución de Portico MCP

## Objetivo del producto

La instalación es declarativa y orientada al terminal. El operador controla:

~~~text
.env                   # incluye VPS_AGENT_SCOPE_ROOT
config/policy.yaml     # policy lógica de capacidades/recursos
~~~

Ambos son estado local del operador. Docker Compose inicia el paquete; no existe un instalador paralelo que oculte la configuración.

## Frontera soportada

El usuario ejecuta los comandos directamente en la VPS. No se requiere un desarrollador, shell remoto controlado por ChatGPT ni revelar contraseña de VPS, clave SSH privada, acceso administrativo irrestricto o secretos ajenos al servicio.

La contraseña del operador OAuth sí es una credencial del servicio: se introduce localmente en `setup-integrated-auth.sh`, sin eco, y no se entrega a ChatGPT.

Consulta [installation-contract.md](installation-contract.md).

## Punto de entrada guiado

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

`install.sh` orquesta los componentes transparentes existentes, muestra la autoridad efectiva antes de iniciar el runtime, pausa solo ante decisiones inevitables y puede volver a ejecutarse tras una interrupción.

## Fase 1 — Bootstrap

El perfil interactivo recomendado es **Standard**: techo físico `/opt` y ninguna raíz de proyecto estática. El techo es el límite máximo del filesystem, no una autorización de lectura/escritura. El operador puede elegir otro techo absoluto.

Equivalente avanzado:

~~~bash
bash scripts/init.sh --scope /opt --dynamic-baseline
~~~

Antes de cambiar estado, `scripts/preflight.py` comprueba host, runtime, Docker, Compose y la situación de 80/443. Los requisitos obligatorios faltantes detienen el proceso antes de crear `.env`, policy o estado.

Después se crean secretos locales aleatorios, ID estable de instancia, `config/policy.yaml`, el techo elegido, usuario real no-root para shell confinado, directorio de estado y validación de Compose.

## Techo físico del filesystem

`compose.yaml` monta únicamente `VPS_AGENT_SCOPE_ROOT` en el Broker bajo `/host`. El Broker exige que rutas de filesystem, cwd de shell y Compose permanezcan dentro de ese techo; cualquier escape falla cerrado.

Cambiar `VPS_AGENT_SCOPE_ROOT` cambia el techo físico, no las raíces lógicas ya autorizadas. Una instalación Standard nueva usa `--dynamic-baseline`, por lo que elegir `/opt` no autoriza `/opt`.

Whole Host requiere `VPS_AGENT_WHOLE_HOST=1` y `compose.host.yaml`, llevando el techo físico a `/`. Eso no activa Full.

## Fase 2 — Autoridad MCP

En Standard no se autoriza ningún proyecto durante la instalación.

`permissions.discover_scope` puede mostrar solo los nombres de los directorios inmediatamente debajo del techo. No abre archivos ni desciende en esas carpetas.

Después de conectar MCP:

~~~text
permissions.request_root_access
 -> solicitud pendiente en el Broker
 -> elicitation MCP / confirmación nativa del cliente
 -> delegación activa ligada al subject
~~~

Perfiles:

- `read`: lectura de filesystem;
- `work`: lectura/escritura + cwd de shell confinado;
- `compose`: añade operaciones Compose que la policy estática ya permita.

En clientes con MCP elicitation, la confirmación es nativa. No se usa el antiguo iframe de aprobación.

### Secretos dentro de proyectos autorizados

Existe una segunda frontera. Por defecto `.env` y `.env.*` siguen bloqueados, mientras `.env.example`, `.env.sample` y `.env.template` siguen siendo plantillas normales.

Para un archivo protegido se requiere `permissions.request_sensitive_access`: permiso temporal, exact-path, ligado al subject, auditable, con expiración y revocación independiente. Los jobs shell Scoped enmascaran estos paths, incluidos aliases por hardlink detectados, salvo que exista permiso temporal explícito.

Una aprobación dinámica puede alcanzar una subcarpeta o el propio techo físico tras una advertencia reforzada. Nunca puede escapar del techo. Autorizar el techo es amplio porque abarca carpetas actuales y futuras, pero aun así no abre secretos protegidos.

systemd, Docker/Compose, red, paquetes, usuarios/grupos, firewall y elevación temporal siguen controlados separadamente por policy.

## Fase 3 — Arranque

~~~bash
docker compose up -d --build
~~~

~~~text
Gateway: no-root, sin root del host
Broker: privilegiado, host montado en /host, sin puerto remoto de control
~~~

Docker empaqueta al Broker; la frontera de autorización es la policy del Broker.

## Fase 4 — Verificación local

~~~bash
bash scripts/verify.sh
~~~

Comprueba Compose, salud de Broker/Gateway, integridad de auditoría, autenticación y una llamada inocua `system.info`.

Éxito local significa runtime listo, **no instalación completa**.

## Fase 5 — OAuth integrado + endpoint público

El usuario configura un hostname DNS que apunta a la VPS. El instalador explica el registro A/AAAA y valida resolución.

~~~bash
bash scripts/setup-integrated-auth.sh
~~~

Si existe exactamente un Traefik reutilizable, se reutiliza. Si no existe y 80/443 están libres, se inicia el Traefik incluido. Después se inician ZITADEL + PostgreSQL, se crea el operador dedicado no-admin, se muestran usuario y correo OAuth, se crea la audiencia de recurso MCP y el cliente privado de introspección, se habilita Dynamic Client Registration compatible con MCP, se liga la identidad Gateway/Broker y se ejecuta la verificación pública.

Éxito:

~~~text
INTEGRATED AUTH: READY
~~~

Verificación repetible:

~~~bash
bash scripts/verify-public.sh
~~~

Valida HTTPS, discovery OAuth/OIDC, metadata del recurso protegido, DCR/PKCE, introspección privada y rechazo fail-closed del MCP no autenticado. Nunca sustituye un servidor web desconocido que ya ocupe 80/443.

## Fase 6 — Capacidad de ChatGPT

Antes de la conexión, comprueba que la cuenta/workspace de ChatGPT realmente expone modo desarrollador y creación de apps MCP personalizadas. La disponibilidad es controlada por OpenAI y puede cambiar.

Si la creación de MCP no está disponible, el servidor puede estar sano pero la puerta final no puede completarse con esa cuenta.

## Fase 7 — Tutorial de ChatGPT Web

~~~bash
bash scripts/connect-chatgpt.sh
~~~

La creación/selección de la app y el login OAuth ocurren en ChatGPT Web. El script muestra endpoint y credenciales de identidad OAuth dedicadas. Nunca usa credenciales VPS/SSH.

~~~text
Tutorial mostrado.
La instalación todavía NO está completa.
~~~

## Fase 8 — Verificación real

El script fija primero una línea base de auditoría. El usuario completa ChatGPT a su ritmo y vuelve al terminal para pulsar Enter.

Se pide a ChatGPT llamar `system.info`. Cada Enter busca una llamada posterior a la línea base. Si no existe, Portico explica qué revisar y permite reintentar sin reiniciar.

Éxito exige:

~~~text
llamada MCP real de ChatGPT
+ subject autenticado esperado
+ policy permite
+ ejecución correcta
+ registro de auditoría del Broker
+ cadena de auditoría válida
~~~

Solo entonces:

~~~text
CHATGPT WEB CONNECTION VERIFIED
INSTALLATION COMPLETE
~~~

## Desarrollo

Probes temporales, runners efímeros/self-hosted, diagnósticos ad hoc y shells de desarrollo **no son pasos de instalación**. `scripts/check-installation-contract.py` protege esta frontera en CI.

## Actualización

~~~bash
bash scripts/update.sh
~~~

Hace backup de `.env`, policy y estado, aplica actualización Git fast-forward, reconstruye, verifica y revierte código/estado si falla.

## Eliminación

Segura:

~~~bash
bash scripts/remove.sh safe
~~~

Purge explícito:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

También eliminar el checkout:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Portico nunca borra directorios de proyecto delegados, aplicaciones, servicios, contenedores, bases de datos o archivos de terceros solo por haber tenido autorización para gestionarlos.
