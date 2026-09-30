# Architektur

## Ziel

Einem KI-Client nützlichen operativen Zugriff auf eine Linux-VPS geben, während die VPS, nicht das Modell, Autorisierung und Privileg kontrolliert.

> **Das LLM ist niemals die Sicherheitsgrenze.**

## Kanonische Runtime

Zwei projekt-eigene Go-Prozesse:

~~~text
ChatGPT / MCP client
        |
        | MCP Streamable HTTP
        v
+-----------------------------+
| vps-agent-gateway           |
| unprivileged                |
| MCP + auth + schemas        |
+-------------+---------------+
              |
              | Unix Domain Socket
              v
+-------------+---------------+
| vps-agent-broker            |
| privileged, local-only      |
| authoritative policy        |
| SQLite + locks + jobs       |
| secrets + audit             |
| files + systemd + Docker    |
| sandboxed execution         |
+-------------+---------------+
              |
              v
     Linux / systemd / Docker
~~~

Reverse Proxy oder privater Tunnel ist Ingress-Infrastruktur, kein dritter Projektdienst.

## Warum Go

Offizielles MCP Go SDK ist Tier 1 und unterstützt MCP 2026-07-28. Eine Sprache reduziert Packaging/Dependencies/Wartung bei gleicher Prozess-Privileggrenze.

## Gateway

Läuft ohne root. Darf MCP anbieten, OAuth/OIDC und Schemas validieren, Tools/Resources normalisieren, nicht-autoritative Preflights durchführen und Broker per Unix Socket aufrufen.

Darf nicht als root laufen, Docker socket erhalten, privilegierte SQLite DB öffnen, Plaintext Secrets lesen, Autorisierungsinstanz werden oder eigene Elevation genehmigen.

## Broker

Privilegierte Sicherheitsgrenze. Jede Aktion wird erneut autorisiert gegen:

~~~text
subject
+ canonical tool
+ canonical resource
+ action
+ current policy
+ grant/lease/job wenn nötig
~~~

Broker besitzt Policy Evaluation, SQLite, Idempotency, Locks, Jobs, sichere Filesystem-Operationen, Docker/systemd, Sandbox, Secret Resolution und Audit. Gateway gilt als untrusted deputy.

## Capability-Modell

Ein breiter Katalog, effektive Autorität durch Policy. Eine Root, mehrere Roots oder Whole Host möglich.

Physische Obergrenze und logische Roots sind getrennt: Ceiling ist maximale Grenze, kein Read Grant. Standard darf nur unmittelbare Verzeichnisnamen discovery-only zeigen. Dynamische Delegationen gehören Broker-State, sind Subject-gebunden, haben `read`/`work`/`compose` und optional TTL.

Pending Request ist keine Autorisierung. MCP Elicitation lässt den Client native Human Confirmation rendern. Broker validiert Request, Subject und one-time token. Confirmation Tools sind nicht model-visible. Ohne Elicitation fail-closed zum Operator-Fallback. Revocation wirkt ohne Restart.

Secret-Dateien sind verschachtelte Grenze: `.env` braucht separate temporäre Exact-Path-Freigabe. Templates bleiben normal. Shell-Sandbox maskiert geschützte Pfade.

Filesystem ist nur eine Dimension; systemd, Docker, Shell, Netzwerk, Admin sind separat. Siehe [product-model.md](product-model.md).

## Policy / Approval

Policy Engine ist Modul im Broker, kein eigener Daemon. Policies deny-by-default. Presets: Controlled, Scoped, Full; Full standardmäßig aus.

Routine Scoped ohne Unterbrechung. Temporäre Elevation wird außerhalb des MCP-Aktionskanals bestätigt. Native Elicitation ist bevorzugt, Broker bindet Approval an Subject + one-time nonce und verhindert Replay/Autoapproval. Separater Admin-Pfad bleibt Fallback/MFA.

## Full

Explizite Capabilities:

~~~text
shell.admin
filesystem.read:any
filesystem.write:any
docker.admin
systemd.admin
~~~

Unrestricted Network ist separat. Full temporär, revokabel, gated.

## Transport

MCP Streamable HTTP an stabilem HTTPS-Endpunkt. Core stateless; keine eigenen WebSocket-/Sessionmechanismen. Jobs/Leases mit Handles, SDK übernimmt Negotiation.

## Filesystem

Keine String-Prefix-Autorisierung. Bevorzugt `openat2` mit restriktiven Flags. Fallback sicherer Directory-FD Walk mit `openat/fstatat/O_NOFOLLOW` oder fail closed. Nie still auf Stringchecks zurückfallen.

## Process Isolation

systemd transient units, cgroups, NoNewPrivileges, PrivateTmp, Filesystem Restrictions, MemoryMax, TasksMax, Deadline, Output Limit, Cancellation. Landlock nur Defense-in-Depth.

## Docker

Gateway bekommt nie Docker socket. Typed Broker Operations gegen kanonische Stacks/Actions.

## Jobs

~~~text
job.start
job.status
job.tail
job.cancel
~~~

Durable internes Modell, unabhängig von HTTP. MCP Tasks kann später adaptiert werden.

## State

SQLite nur Broker, kurze Transaktionen. State für approvals, leases, jobs, idempotency journal, locks, audit metadata.

Resource Lock mindestens mit Resource, Owner/Action-ID, monotonic fencing token und expires_at. Alte Owner dürfen neuere Tokens nicht freigeben.

Idempotency Journal:

~~~text
PENDING -> external effect -> DONE
~~~

PENDING vor Side Effect committen; Crash danach führt zu Reconciliation statt Blind Retry.

## Secrets

Native Linux bevorzugen: root-owned files außerhalb Repo, systemd credentials. Broker löst refs und injiziert nur in Zielprozesse. Gateway/model-facing Tools liefern kein Plaintext.

## Audit

Gate 1: structured local audit.
Scoped production: durable sequence/integrity.
Vor Full: tamper-evident hash chain, remote checkpoint/forwarding, getestetes Recovery.

## Downstream Tool Trust

Upstreams out-of-band allowlisted, deterministic namespaced names, Schemaänderungen fingerprinted/reviewed, Ergebnisse untrusted data. Ergebnisse ändern nie Policy, Leases, Serverregistrierung oder Secret Exposure.

## Nichtziele

Kein universal MCP Gateway, Multi-Tenant Control Plane, Distributed Scheduler, Generic Root Shell, Kubernetes-Projekt, SSH-Ersatz oder Every-Client-Framework.

> Erstes Ziel: sicher beweisen, dass der echte KI-Client eine kleine, nützliche, auditierbare Aktion auf einer VPS ausführen kann.
