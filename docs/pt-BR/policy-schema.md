# Schema de policy

Policies são documentos de capacidades deny-by-default interpretados de forma autoritativa pelo Broker.

Campos desconhecidos, capacidades desconhecidas e valores enum inválidos falham fechados.

## Exemplo Controlled

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

Habilite somente as capacidades exigidas pelo stack real.

## Preset Full

Full é desabilitado por padrão.

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

Full é um preset de conveniência que expande para capacidades explícitas. `network.unrestricted` nunca é adicionado implicitamente.

## Regras de validação

- chaves desconhecidas: rejeitar;
- capacidades desconhecidas: rejeitar;
- enums desconhecidos: rejeitar;
- raízes de path inválidas: rejeitar;
- raízes de escrita sem permissão explícita: rejeitar;
- wildcards administrativos exigem Full habilitado e grant humano válido;
- rede irrestrita exige aprovação explícita separada;
- TTL acima do máximo do servidor: rejeitar;
- ativação de policy estática exige reload/ativação autoritativa do Broker;
- editar o arquivo de policy via agente não o ativa automaticamente;
- delegações dinâmicas devem permanecer dentro do teto físico;
- delegação dinâmica muda escopo de recurso, não as listas estáticas de ações;
- expansão dinâmica de permissão exige solicitação pendente + aprovação do operador;
- revogação dinâmica pode ocorrer in-band porque reduz autoridade;
- expiração de delegação temporária é aplicada pelo estado do Broker.
