# Policy schema

Policies are deny-by-default capability documents. Unknown fields and unknown capabilities should fail validation unless a future schema version explicitly allows them.

## Scoped minimum schema

```yaml
version: 1
mode: scoped

filesystem:
  read: []
  write: []

network:
  mode: blocked | allowlist | unrestricted
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
  enabled: true
  cwd_roots: []
  max_runtime_seconds: 300
  max_output_bytes: 1048576

privilege:
  admin: deny | broker-only

replay:
  require_idempotency_for_safe_writes: true
  blind_retry_non_replay_safe: false
```

## Full preset

`mode: full` is only a UI/policy convenience preset. It expands into an explicit capability set, for example:

```yaml
mode: full
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

lease:
  required: true
  max_ttl_minutes: 60
```

`network.unrestricted` must never be implicitly added by Full.

## Validation rules

- unknown keys: reject
- unknown enum values: reject
- write roots without an explicit mode: reject
- wildcard administrative capabilities require a human-approved lease
- unrestricted network requires explicit separate approval
- TTL above server maximum: reject
- policy changes never take effect merely because the model edited a file; an authoritative reload path must validate and activate them
