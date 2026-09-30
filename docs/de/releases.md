# Release- und Versionsrichtlinie

Portico MCP folgt Semantic Versioning.

## Channels

### Stable

Stabile Releases verwenden Tags wie `v0.1.0`, `v0.1.1` und `v0.2.0`. Sie sind der normale Updatekanal; `main` ist nicht Produktion.

### Release Candidate

RCs verwenden SemVer-Pre-Release-Tags wie `v0.1.0-rc.1` für die finale Abnahme.

### Entwicklung

`main` ist der Entwicklungsbranch und nicht Standardziel von `scripts/update.sh`.

## RC5

Aktueller Kandidat ist `0.1.0-rc.5`. RC5 bringt Ceiling Discovery ohne Inhaltszugriff, Secret-Schutz in delegierten Wurzeln, native MCP Elicitation, Sicherheitsannotationen und GitHub-native manuelle Release-Promotion. Frühere RC-Tags bleiben unveränderliche Historie.

Stable `v0.1.0` wird erst nach dem finalen Acceptance-Gate auf sauberer Installation erstellt.

## Veröffentlichung

Bevorzugt **GitHub Actions → release → Run workflow** auf `main`. Erst nach Source-, Test- und Installationsvertragsprüfung werden exakter Tag, Multi-Arch-Images und GitHub Release erzeugt.

## Artefakte

Gateway/Broker Images für `linux/amd64` und `linux/arm64`, Docker-Compose-Bundle, SHA-256-Checksums und Release Notes. Security CI erzeugt Reports und CycloneDX SBOM.

## Update

`scripts/update.sh` wählt standardmäßig den neuesten stabilen Tag von `origin`, verweigert non-fast-forward und behält Backups.

~~~bash
VPS_AGENT_UPDATE_REF=v0.1.0-rc.5 bash scripts/update.sh
~~~
