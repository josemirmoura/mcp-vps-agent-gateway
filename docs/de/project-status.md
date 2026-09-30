# Projektstatus und Reife

## Aktueller Stand

**PRE-RELEASE PRODUCTIZATION / CHATGPT E2E GATE COMPLETE**

Die Docker-first-Go-Implementierung hat wiederholbare Clean-Runner-Evidenz für Scoped Runtime, Package Lifecycle und Host-Operationen.

Noch fehlen Stable Production Release, Produktions-Kompatibilitätsversprechen, langfristige Real-VPS-Historie und Produktionsclaim für Full/Admin-Shell.

Labor-Evidenz deckt Filesystem CRUD, Denials, sandboxed Shell/Jobs, Host-systemd, Host Docker/Compose, Diagnostik, Lifecycle und Audit ab. Am 2026-09-28 wurde der unterstützte integrierte OAuth-Pfad mit ChatGPT Web auf einer realen VPS durch einen authentifizierten `system.info`-Aufruf getestet, vom Broker gesehen und in der Audit-Kette aufgezeichnet. Gate 0A ist damit geschlossen, ohne Stable-Produktionsclaim.

## Reifeleiter

### R0 — Nur Architektur

Dokumentation, keine ausführbare Referenzimplementierung.

### R1 — Produktpfad + Gate 0B

Gate 0A für integriertes OAuth + ChatGPT Web abgeschlossen; MCP Inspector und sichere POC in Wegwerf-Testroot.

### R2 — Typisierter privilegierter Pilot

Getrennte Gateway/Broker-Prozesse, eine nicht kritische Service-Aktion über typed tools, fail-closed und lokales Audit.

### R3 — Scoped Pilot

Ein echter Stack unter expliziter Policy, mehrere Tage getestet, Recovery/Denials geübt, kein routinemäßiges Full.

### R4 — Gehärtete Scoped-Produktion

Deployment-Auth, dauerhafter Broker-State, Jobs/Retry/Locks, Secret Delivery, Backup/Recovery und Monitoring getestet. Erst dann production-capable für dokumentiertes Scoped.

### R5 — Elevated/Full-Produktion

Zusätzlich: Full-Flag, out-of-band Approval, temporäre Grants/Expiry, revoke-all, separate Network Elevation, tamper-evident Audit + Remote Checkpointing und getestetes Admin-Recovery.

## Full-Default

~~~yaml
features:
  full_mode_enabled: false
~~~

Full ist kein Erfolgskriterium. Eine starke Scoped-Implementierung ist ein gültiges Produktionsziel.

## Evidenz statt Popularität

Stars/Forks sind Community-Signale, keine Produktionsbeweise. Bevorzuge reproduzierbare Builds, Tests, Releases, Deployment-Historie, Recovery-Tests, Incident Learnings, Dependency Maintenance, Security Review und externe Review.

## Nächster Meilenstein

**Owner Clean-Install Acceptance von `v0.1.0-rc.5`**. RC5 umfasst native MCP Elicitation, discovery-only Ceiling Inventory, Secret Enforcement und Safety Annotations.

Blocker für Stable `v0.1.0`: exakter RC5-Tag auf sauberer Zielumgebung, echtes ChatGPT OAuth, auditiertes `system.info`, native Approval UX, Discovery ohne Inhaltsleck, repräsentative Scoped-Operationen, `.env` Denial/temporäre Ausnahme/Revocation, Root-Revocation und Lifecycle. Langzeitreliabilität bleibt später. Siehe [operator-acceptance.md](operator-acceptance.md) und [implementation-validation.md](implementation-validation.md).
