# Modelo de producto: toolbox completo, autoridad Scoped

## Decisión central

Distribuir un catálogo completo y controlar autoridad mediante policy server-side.

No hay binarios separados limited/project/full:

~~~text
mismos binarios
+ mismo catálogo MCP
+ policy distinta
= autoridad efectiva distinta
~~~

El LLM nunca decide el scope.

## Scope del usuario

El operador decide cuánto de la VPS delega. Ejemplos no crean autoridad; la policy explícita confirmada sí.

~~~text
una carpeta:
/opt/my-app

varias:
/opt/app-a
/var/www/site
/srv/data

filesystem completo:
/
~~~

Lo mismo aplica por separado a systemd, Docker, network, packages, users/groups y recursos administrativos.

## Perfiles

### Standard

Recomendado en hosts multi-proyecto. Techo físico por defecto `/opt`, sin raíz autorizada durante instalación.

Portico descubre solo nombres de carpetas inmediatas. Contenido bloqueado hasta aprobar raíz con `read`, `work` o `compose`. Secretos protegidos siguen como frontera interna.

### Project

Acceso autónomo dentro de una raíz, por ejemplo `/opt/my-app`: CRUD, shell Scoped con cwd dentro, Docker seleccionado, systemd seleccionado y destinos de red seleccionados.

### Policy avanzada/estática

Operadores avanzados pueden definir varias raíces y grupos en policy. Standard guiado prefiere delegación runtime y evita exigir YAML a usuarios nuevos.

### Whole host

Autoriza explícitamente recursos host-wide. Es policy, no build diferente. Puede incluir `/`, systemd amplio, Docker, packages, users/groups, firewall/network y shell admin temporal. Todo sigue explícito/auditable.

## Catálogo de capacidades

### Filesystem

`file.list/stat/read/mkdir/write/patch/copy/move/remove/remove_recursive/hash/chmod/chown`.

Las capabilities existen aunque policy las deshabilite. Recursive delete y permission changes amplios son capabilities separadas.

### Commands/jobs

`shell.exec`, `job.start/status/tail/cancel`.

`shell.exec` debe correr en sandbox server-side. YAML por sí solo no confina un comando. La policy se traduce a cwd roots, ProtectSystem, ReadWritePaths/ReadOnlyPaths, ProtectHome, PrivateTmp, cgroups, MemoryMax, TasksMax, timeout, output limits y network policy.

### systemd

`service.list/status/logs/start/stop/restart/reload/enable/disable`, limitado por unidades/acciones canónicas.

### Docker/Compose

`docker.list/inspect/logs/start/stop/restart`, `compose.config/pull/up/down`.

Prefiere operaciones tipadas. Docker socket nunca llega al Gateway.

### Diagnóstico

`system.info/health/disk/memory`, `process.list/inspect`, `network.listen/check`, `journal.read`.

### Administración

Disponibles pero deshabilitadas salvo selección explícita: package update/install/remove, users/groups, firewall, chmod/chown fuera de rangos seguros y `shell.exec_admin`.

## Scope multidimensional

Filesystem es un eje entre varios: roots, systemd, Docker stacks, shell cwd, network destinations, package actions, users/groups, firewall y admin capabilities.

Un Project puede tener CRUD completo en `/opt/my-app` y cero autoridad sobre nginx, Docker, apt o Internet.

## chmod y destrucción

`file.chmod`, incluso 0777, puede existir si policy permite. Defaults no habilitan world-writable silenciosamente. `file.remove_recursive` es destructiva separada; host-wide chmod/chown es decisión administrativa.

## UX de configuración

`config/policy.yaml` decide roots/recursos, preset, filesystem, shell/sandbox, systemd, Docker/Compose, network, administración, approval/elevation y autenticación/exposición. Es autoritativo y legible por humano/IA. Compose/runtime rechazan configuración inválida y puede editarse sin reinstalar.

## Packaging

Docker Compose oficial:

~~~text
Gateway container
  non-root
  no /host
  no Docker socket
        |
        | Unix socket
        v
Broker container
  privileged host-control boundary
  host mounted at /host
        |
        v
VPS
~~~

Broker no es sandbox alrededor del host; es la frontera privilegiada empaquetada. Docker da packaging/lifecycle; Gateway queda no-root; Broker recibe host root solo para acciones autorizadas, no expone API remota, y policy decide recursos. Montar `/host` no concede autoridad al LLM.

Instalación primaria:

~~~bash
bash scripts/install.sh
~~~

Orquesta bootstrap, Compose, verify, OAuth y conexión ChatGPT, mostrando autoridad y pudiendo retomarse. Comandos individuales siguen disponibles. [installation-contract.md](installation-contract.md) define la frontera soportada.

## Instalación termina solo tras ChatGPT

El tutorial no basta. Después de runtime/policy debe mostrar endpoint, autenticación, scope efectivo y siguiente paso.

La primera ejecución:

1. verifica HTTPS;
2. verifica auth;
3. ejecuta test server-side inocuo;
4. muestra autoridad;
5. presenta tutorial ChatGPT vigente;
6. espera conexión;
7. exige tool call inocua desde ChatGPT;
8. verifica subject, policy, ejecución y audit;
9. solo entonces marca completo.

La UI de ChatGPT cambia, así que el tutorial se versiona/revalida.

~~~text
clone / bundle
 -> elegir autoridad
 -> iniciar Gateway + Broker
 -> validar policy
 -> OAuth + HTTPS
 -> tutorial ChatGPT
 -> conectar ChatGPT
 -> verificar llamada E2E
 -> verificar auditoría
 -> instalación completa
~~~
