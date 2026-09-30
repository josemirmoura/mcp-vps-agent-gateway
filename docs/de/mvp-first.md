# MVP-first Implementierungspfad

Die vollständige Architektur ist das Nordziel, nicht der erste Meilenstein.

## Gate -1 — Adopt, adapt oder build

Zuerst bestehende Produkte/Open-Source-MCP-Server prüfen. Adoptieren, wenn passend; adaptieren, wenn nahe; nur bauen, wenn die Kombination fehlt.

Differenzierendes Ziel:

~~~text
ChatGPT Web
+ selbst kontrollierte VPS
+ serverseitige Policy
+ Scoped Autonomie
+ optionale temporäre Elevation
+ Broker unter Operator-Kontrolle
~~~

## Gate 0A — ChatGPT-Produktoberfläche beweisen

**Abgeschlossen 2026-09-28 für integriertes OAuth + ChatGPT Web.**

Der echte Zielclient muss vor dem privilegierten Produktpfad validiert werden. Nicht annehmen, dass jeder Plan private write-fähige MCPs direkt unterstützt.

Routen:

1. Workspace/Plan mit private full MCP write;
2. geeignete App/Plugin-Route mit Remote Write;
3. MCP Inspector solange Distribution ungeklärt ist.

Dokumentieren:

~~~text
target_surface:
target_plan:
integration_route:
read_available:
write_available:
private_or_published:
tested_date:
~~~

Wenn ein zukünftiges Konto Custom MCP nicht anbietet, an dieser Produktgrenze stoppen oder nur die Distribution ändern. Server nicht schwächen.

## Gate 0B — Safe MCP POC

Unprivilegierter Go-Server + offizielles MCP Go SDK. Nur:

~~~text
system.info
file.read_test
file.write_test
~~~

Filesystem auf `/tmp/vps-agent-poc/` begrenzen. Kein root, Docker, systemd write, SQLite, externe Secrets, Full, Approval, generische Shell.

Erfolg: Inspector, Discovery, read/write je nach Client, Forbidden Paths fail, verständliche Fehler, sichere Retry/Reconnect.

## Gate 1 — Eine typed privileged action

Minimaler Broker via Unix Socket. `service.status` und `service.restart` nur für eine disposable/non-critical Unit. Generic admin shell, Full, Approval UI, breite Docker-Administration und Remote Audit bleiben draußen.

## Gate 2 — Scoped Real-Stack Pilot

Explizite Policy für einen echten Stack. Nur erwiesene Capabilities hinzufügen, z. B. `file.read`, Service Status/Restart, Docker Logs/Restart und Job Status. Mehrere Tage messen: Frequenz, Erfolg, Korrekturen, False Denials, fehlende Capabilities, Recovery. Bestehen wenn Scoped nützlich ist ohne Routine-Elevation.

## Gate 3 — Durability

Bei Bedarf Broker-SQLite, durable jobs, Idempotency Identity, Logical Locks, systemd Credentials/Secret Refs und stärkeres strukturiertes Audit.

## Gate 4 — Breitere Writes

Nach Bedarf `file.write`/`file.patch`, validierte Config Updates, weitere typed Docker/systemd Actions und sandboxed `shell.exec` in Scoped Roots. Generische Shell ist kein Muss.

## Gate 5 — Temporary Elevation

Nur wenn Scoped nachweislich nicht reicht: Elevation Requests, out-of-band human approval, temporary leases, revoke-all, Remote Audit Anchoring, Full Flag. Erst danach `shell.exec_admin` erwägen.

## Regel

Jede neue Komponente braucht Evidenz aus dem vorherigen Gate.

> Kann der echte Zielclient wertvolle Arbeit auf einem echten Server sicher und zuverlässig erledigen?
