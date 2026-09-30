# Policy schema

Policy は Broker が authoritative に解釈する deny-by-default capability document です。

Unknown field、unknown capability、invalid enum は fail-closed で reject します。

## Controlled example

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

## Scoped minimum shape

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

Real stack に必要な capability だけを有効化します。

## Full preset

Full は default disabled。

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

Full は explicit capability に展開される convenience preset です。`network.unrestricted` は implicit に追加されません。

## Validation rules

- unknown keys/capabilities/enums: reject;
- invalid path roots: reject;
- write roots without explicit permission: reject;
- administrative wildcard requires Full feature + valid human grant;
- unrestricted network requires separate approval;
- TTL above server maximum: reject;
- static policy activation requires authoritative Broker reload/activation;
- agent が policy file を edit しても自動 activation しない;
- dynamic root delegation は physical ceiling 内;
- dynamic delegation は resource scope だけを変更し static action list は変えない;
- permission expansion は pending request + operator approval;
- revocation は authority reduction のため in-band 可能;
- temporary expiry は Broker state が enforce。
