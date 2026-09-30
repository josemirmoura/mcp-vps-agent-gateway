# Dokumentationsübersicht

Die kanonische Dokumentation liegt in `docs/`. Dieser Ordner ist die offizielle deutsche Übersetzung. Falls eine Übersetzung von der englischen Datei derselben Version abweicht, gilt Englisch bis zur Korrektur als technische Referenz.

## Operator / Benutzer

1. [quick-start.md](quick-start.md) — kürzester unterstützter Installationsweg.
2. [installation-contract.md](installation-contract.md) — normative Installationsgrenze.
3. [installer-flow.md](installer-flow.md) — vollständiger Installationsablauf.
4. [product-model.md](product-model.md) — Project, Standard und Whole Host.
5. [chatgpt-integration.md](chatgpt-integration.md) — ChatGPT-Verbindung und Abschluss-Gate.
6. [authentication.md](authentication.md) — integriertes OAuth/OIDC.
7. [operations.md](operations.md) — Status, Logs, Audit, Update, Entfernung.
8. [troubleshooting.md](troubleshooting.md) — Fehlerbehebung.
9. [faq.md](faq.md) — häufige Fragen.
10. [compatibility.md](compatibility.md) — validierte und nicht validierte Umgebungen.
11. [privacy.md](privacy.md)
12. [releases.md](releases.md)
13. [support.md](support.md)
14. [operator-acceptance.md](operator-acceptance.md)

## Entwicklung / Sicherheit

- [project-status.md](project-status.md)
- [architecture.md](architecture.md)
- [policy-schema.md](policy-schema.md)
- [threat-model.md](threat-model.md)
- [security-hardening-v2.md](security-hardening-v2.md)
- [security-release.md](security-release.md)
- [runtime-semantics-and-recovery.md](runtime-semantics-and-recovery.md)
- [tool-trust-and-confused-deputy.md](tool-trust-and-confused-deputy.md)
- [transport-and-aggregation.md](transport-and-aggregation.md)
- [implementation-validation.md](implementation-validation.md)
- [simulation-validation.md](simulation-validation.md)
- [mvp-first.md](mvp-first.md)
- [implementation-runbook.md](implementation-runbook.md)
- [build-vs-adopt.md](build-vs-adopt.md)
- [multi-instance.md](multi-instance.md)
- [productization-status.md](productization-status.md)
- [references.md](references.md)

Bei Konflikten gilt: Architektur, Installationsvertrag, aktueller Projektstatus, Policy-Schema, Security-Hardening, Runtime-Semantik, unterstützende Dokumente, historische Dokumente.

**Architektur in einem Satz:** zwei Go-Prozesse, ein unprivilegiertes MCP Gateway und ein privilegierter lokaler Broker über Unix-Socket; der Broker kontrolliert Autorisierung, Zustand und privilegierte Ausführung.
