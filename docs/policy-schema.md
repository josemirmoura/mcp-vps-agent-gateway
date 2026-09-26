# Policy schema

Policies are deny-by-default capability documents interpreted authoritatively by the Broker.

Unknown fields, unknown capabilities and invalid enum values fail closed.

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

Enable only the capabilities a real stack needs.

## Full preset

Full is disabled by default.

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

Full is a convenience preset that expands into explicit capabilities.

network.unrestricted is never implicitly added.

## Validation rules

- unknown keys: reject
- unknown capabilities: reject
- unknown enum values: reject
- invalid path roots: reject
- write roots without explicit permission: reject
- wildcard administrative capabilities require Full feature enabled plus a valid human-approved grant
- unrestricted network requires separate explicit approval
- TTL above server maximum: reject
- policy activation requires an authoritative Broker reload/activation path
- editing a policy file through an agent does not automatically activate it
