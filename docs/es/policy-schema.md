# Schema de policy

Las policies son documentos de capacidades deny-by-default interpretados autoritativamente por el Broker.

Campos desconocidos, capacidades desconocidas y enums inválidos fallan cerrados.

## Ejemplo Controlled

~~~yaml
version: 1
mode: controlled

filesystem:
  read:
    - /srv/app/**
  write:
    - /tmp/vps-agent/**

network:
  mode: blocked
  destinations: []

services:
  inspect:
    - "*"
  manage: []
  actions: []

docker:
  inspect: []
  manage: []
  actions: []

shell:
  enabled: false
  cwd_roots: []
  max_runtime_seconds: 120
  max_output_bytes: 1048576

privilege:
  admin: deny
~~~

## Forma mínima Scoped

~~~yaml
version: 1
mode: scoped

filesystem:
  read: []
  write: []

network:
  mode: blocked
  destinations: []

services:
  inspect: []
  manage: []
  actions: []

docker:
  inspect: []
  manage: []
  actions: []

shell:
  enabled: false
  cwd_roots: []
  max_runtime_seconds: 300
  max_output_bytes: 1048576

privilege:
  admin: broker-only

replay:
  require_idempotency_for_safe_writes: true
  blind_retry_non_replay_safe: false
~~~

Habilita solo las capacidades necesarias.

## Preset Full

Full está deshabilitado por defecto.

~~~yaml
version: 1
mode: full
enabled: false

capabilities:
  - shell.admin
  - filesystem.read:any
  - filesystem.write:any
  - docker.admin
  - systemd.admin

network:
  mode: allowlist
  destinations: []
  unrestricted_requires_separate_approval: true

grant:
  required: true
  max_ttl_minutes: 60
~~~

Full es un preset que expande a capacidades explícitas. `network.unrestricted` nunca se añade implícitamente.

## Reglas de validación

- claves desconocidas: rechazar;
- capacidades desconocidas: rechazar;
- enums desconocidos: rechazar;
- raíces de path inválidas: rechazar;
- raíces de escritura sin permiso explícito: rechazar;
- wildcards administrativos requieren Full habilitado y grant humano válido;
- red irrestricta requiere aprobación separada;
- TTL por encima del máximo: rechazar;
- activar policy estática exige reload/activación autoritativa del Broker;
- editar policy mediante un agente no la activa;
- delegaciones dinámicas deben quedar dentro del techo físico;
- una delegación cambia scope de recurso, no listas estáticas de acciones;
- ampliar permisos requiere solicitud pendiente + aprobación del operador;
- revocar puede hacerse in-band porque reduce autoridad;
- la expiración temporal la aplica el state del Broker.
