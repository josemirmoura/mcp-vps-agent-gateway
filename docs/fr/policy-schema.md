# Schéma de policy

Les policies sont des documents de capacités deny-by-default interprétés autoritativement par le Broker.

Champs inconnus, capacités inconnues et enums invalides échouent fermés.

## Exemple Controlled

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

## Forme minimale Scoped

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

Activez uniquement les capacités réellement nécessaires.

## Preset Full

Full est désactivé par défaut.

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

Full est un raccourci vers des capacités explicites. `network.unrestricted` n’est jamais ajouté implicitement.

## Règles de validation

- clés, capacités et enums inconnus : rejeter ;
- racines de path invalides : rejeter ;
- racines d’écriture sans permission explicite : rejeter ;
- wildcards administratifs exigent Full activé + grant humain valide ;
- réseau sans restriction exige approbation séparée ;
- TTL au-dessus du maximum serveur : rejeter ;
- policy statique exige reload/activation autoritative Broker ;
- éditer le fichier via un agent ne l’active pas ;
- délégations dynamiques restent dans le plafond physique ;
- elles modifient le scope de ressource, pas les listes statiques d’actions ;
- extension d’autorité exige demande pending + approbation opérateur ;
- révocation peut être in-band car elle réduit l’autorité ;
- expiration temporaire est imposée par l’état Broker.
