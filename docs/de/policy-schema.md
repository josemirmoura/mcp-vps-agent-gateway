# Policy-Schema

Policies sind deny-by-default Capability-Dokumente, die vom Broker autoritativ interpretiert werden.

Unbekannte Felder, Capabilities und ungültige Enum-Werte schlagen fail-closed fehl.

## Controlled-Beispiel

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

## Minimale Scoped-Form

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

Nur tatsächlich benötigte Capabilities aktivieren.

## Full-Preset

Full ist standardmäßig deaktiviert.

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

Full expandiert in explizite Capabilities. `network.unrestricted` wird nie implizit hinzugefügt.

## Validierungsregeln

- unbekannte Keys/Capabilities/Enums: ablehnen;
- ungültige Path-Wurzeln: ablehnen;
- Write-Wurzeln ohne explizite Erlaubnis: ablehnen;
- administrative Wildcards benötigen Full-Feature + gültigen menschlichen Grant;
- uneingeschränktes Netzwerk braucht separate explizite Freigabe;
- TTL über Servermaximum: ablehnen;
- statische Policy braucht autoritativen Broker-Reload/-Aktivierung;
- Agentenbearbeitung einer Datei aktiviert sie nicht automatisch;
- dynamische Delegationen bleiben innerhalb der physischen Obergrenze;
- dynamische Delegation ändert Resource-Scope, nicht statische Action-Listen;
- Permission-Erweiterung braucht Pending Request + Operator-Freigabe;
- Revocation darf in-band erfolgen, weil Autorität sinkt;
- Ablauf temporärer Delegationen wird durch Broker-State erzwungen.
