# Inicio rápido de Portico MCP

Portico MCP se instala desde terminal con Docker Compose y scripts transparentes del repositorio.

## Requisitos

- VPS Linux; Ubuntu 24.04 LTS es el objetivo validado del RC;
- Docker Engine 24+ y Docker Compose v2;
- al menos 2 GB RAM para la ruta ZITADEL incluida;
- Git, OpenSSL, Python 3 y curl;
- DNS público, TCP 80/443 y HTTPS válido;
- cuenta/workspace de ChatGPT que realmente permita modo desarrollador y apps MCP personalizadas.

La disponibilidad en ChatGPT es controlada por OpenAI y puede cambiar.

## Empezar

~~~bash
git clone https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

El instalador detecta el idioma del terminal. Idiomas oficiales:

~~~text
en · pt-BR · es · de · fr · ja · id
~~~

También puede elegirse explícitamente:

~~~bash
bash scripts/install.sh --lang es
~~~

## Flujo guiado

~~~text
Requisitos
 -> Scope
 -> Autoridad efectiva
 -> Contenedores
 -> Verificación local
 -> Acceso público seguro
 -> Conectar ChatGPT
 -> llamada MCP real y auditada
 -> INSTALACIÓN COMPLETA
~~~

El preflight comprueba Linux/runtime, Docker/Compose, herramientas, systemd, memoria y proxy de borde. Si falta algo obligatorio, se detiene con una referencia oficial.

Un Traefik existente se reutiliza cuando puede identificarse de forma segura; si no existe y 80/443 están libres, se usa el Traefik incluido.

## Modelo recomendado: Standard

~~~text
techo físico:       /opt
raíces estáticas:   ninguna
autoridad proyecto: aprobada después, bajo demanda
~~~

El techo define hasta dónde podría llegar Portico. **No autoriza /opt** en la instalación.

Puedes elegir otro techo absoluto:

~~~bash
bash scripts/install.sh --profile custom --scope /srv/apps
~~~

Después de conectar ChatGPT, Portico puede usar `permissions.discover_scope` para mostrar solo nombres de carpetas directamente bajo el techo. Para entrar:

~~~text
permissions.request_root_access
 -> confirmación nativa MCP
 -> Broker activa read / work / compose
~~~

La IA no puede autoaprobar su ampliación de autoridad.

### Secretos

Incluso dentro de un proyecto autorizado, `.env` y `.env.*` permanecen bloqueados. Plantillas como `.env.example` siguen legibles. Un secreto real requiere aprobación temporal separada mediante `permissions.request_sensitive_access`.

## Project

~~~bash
bash scripts/install.sh --profile project --scope /opt/my-app
~~~

Con `--create-scope` el instalador puede crear explícitamente una ruta faltante.

## Whole Host

~~~bash
bash scripts/install.sh --profile whole-host
~~~

Cambia el techo físico a `/`, pero no habilita Full, red irrestricta, paquetes, usuarios o firewall automáticamente.

## Usuario de shell

Los jobs Scoped usan un usuario real no-root. Para elegirlo:

~~~bash
bash scripts/install.sh --run-as deploy
~~~

## OAuth público

El instalador guía DNS y crea una identidad OAuth dedicada, mostrando:

~~~text
Username: vps-operator
Email:    operator@example.com
~~~

La pantalla OAuth usa el username. No son credenciales Linux/SSH/root.

## Finalización en ChatGPT

`scripts/connect-chatgpt.sh` muestra el tutorial completo. Configura la app/OAuth a tu ritmo, envía la prueba `system.info`, vuelve al terminal y pulsa Enter. Si la llamada no aparece todavía, puedes corregir y reintentar.

## Solo validación local

~~~bash
bash scripts/install.sh --profile custom --scope /opt --local-only --yes
~~~

Eso **no** es una instalación completa porque omite OAuth público y ChatGPT.

## Eliminación

~~~bash
bash scripts/remove.sh safe
~~~

Purge:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE bash scripts/remove.sh --purge
~~~

También checkout:

~~~bash
VPS_AGENT_PURGE_CONFIRM=PURGE \
VPS_AGENT_REMOVE_SOURCE_CONFIRM=REMOVE_SOURCE \
bash scripts/remove.sh --purge --remove-source
~~~

Portico no elimina arbitrariamente proyectos ni recursos administrados.

## Releases estables

Tras congelar `v0.1.0`, instalaciones orientadas a producción deben usar un tag estable:

~~~bash
git clone --branch v0.1.0 https://github.com/josemirmoura/mcp-vps-agent-gateway.git
cd mcp-vps-agent-gateway
bash scripts/install.sh
~~~

El checkout Git es intencional: permite update fast-forward, migraciones verificadas, backup y rollback automático.
