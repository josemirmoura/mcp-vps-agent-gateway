# Schema policy

Policy adalah dokumen capability deny-by-default yang ditafsirkan secara authoritative oleh Broker.

Field, capability, dan enum yang tidak dikenal ditolak fail-closed.

## Contoh Controlled

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

## Bentuk minimum Scoped

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

Aktifkan hanya capability yang benar-benar diperlukan.

## Preset Full

Full default-nya nonaktif.

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

Full hanyalah preset yang mengembang menjadi capability eksplisit. `network.unrestricted` tidak pernah ditambahkan implisit.

## Aturan validasi

- key/capability/enum tidak dikenal: reject;
- path root tidak valid: reject;
- write root tanpa izin eksplisit: reject;
- wildcard administratif butuh Full enabled + human grant valid;
- unrestricted network butuh approval terpisah;
- TTL di atas maksimum server: reject;
- aktivasi static policy butuh authoritative Broker reload;
- edit file policy lewat agent tidak otomatis mengaktifkannya;
- dynamic root delegation harus di dalam physical ceiling;
- delegation mengubah resource scope, bukan static action list;
- perluasan permission butuh pending request + operator approval;
- revocation boleh in-band karena mengurangi authority;
- expiry temporary delegation ditegakkan oleh state Broker.
